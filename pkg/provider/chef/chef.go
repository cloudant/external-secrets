/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package chef

import (
	"context"
	"fmt"
	"net/url"

	"github.com/go-chef/chef"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	errChefStore                             = "received invalid Chef SecretStore resource: %w"
	errMissingStore                          = "missing store"
	errMisingStoreSpec                       = "missing store spec"
	errMissingProvider                       = "missing provider"
	errMissingChefProvider                   = "missing chef provider"
	errMissingName                           = "missing Name"
	errMissingBaseURL                        = "missing BaseURL"
	errMissingAuth                           = "cannot initialize Chef Client: no valid authType was specified"
	errMissingSecretKey                      = "missing Secret Key"
	errInvalidClusterStoreMissingPKNamespace = "invalid ClusterSecretStore: missing Chef SecretKey Namespace"
	errFetchK8sSecret                        = "could not fetch SecretKey Secret: %w"
	errChefInvalidURL                        = "unable to parse URL: %w"
	errChefInvalidName                       = "invalid name: allowed values are lowecase letters, numbers, hyphens and underscores"
	errChefClient                            = "unable to create chef client: %w"
)

type Providerchef struct {
	chefClient *chef.Client
}

// https://github.com/external-secrets/external-secrets/issues/644
var _ v1beta1.SecretsClient = &Providerchef{}
var _ v1beta1.Provider = &Providerchef{}

func init() {
	v1beta1.Register(&Providerchef{}, &v1beta1.SecretStoreProvider{
		Chef: &v1beta1.ChefProvider{},
	})
}

func (providerchef *Providerchef) NewClient(ctx context.Context, store v1beta1.GenericStore, kube kclient.Client, namespace string) (v1beta1.SecretsClient, error) {
	chefProvider, err := getChefProvider(store)
	if err != nil {
		return nil, err
	}
	credentialsSecret := &corev1.Secret{}
	objectKey := types.NamespacedName{
		Name:      chefProvider.Auth.SecretRef.SecretKey.Name,
		Namespace: namespace,
	}
	if store.GetObjectKind().GroupVersionKind().Kind == v1beta1.ClusterSecretStoreKind {
		objectKey.Namespace = *chefProvider.Auth.SecretRef.SecretKey.Namespace
	} else {
		return nil, fmt.Errorf(errInvalidClusterStoreMissingPKNamespace)
	}

	err = kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return nil, fmt.Errorf(errFetchK8sSecret, err)
	}

	secretKey := credentialsSecret.Data[chefProvider.Auth.SecretRef.SecretKey.Key]
	if (secretKey == nil) || (len(secretKey) == 0) {
		return nil, fmt.Errorf(errMissingSecretKey)
	}

	client, err := chef.NewClient(&chef.Config{
		Name:    chefProvider.Name,
		Key:     string(secretKey),
		BaseURL: chefProvider.BaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf(errChefClient, err)
	}
	providerchef.chefClient = client
	return providerchef, nil
}

// TO be implemented

// Close closes the client connection.
func (providerchef *Providerchef) Close(ctx context.Context) error {
	return nil
}

// Validate checks if the client is configured correctly
// to be able to retrieve secrets from the provider.
func (providerchef *Providerchef) Validate() (v1beta1.ValidationResult, error) {
	return v1beta1.ValidationResultUnknown, nil
}

// GetAllSecrets syncs multiple chef databag Items into a single Kubernetes Secret, for dataFrom.find.
func (providerchef *Providerchef) GetAllSecrets(ctx context.Context, ref v1beta1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, fmt.Errorf("GetAllSecrets yet to implement")
}

// GetSecret returns a single secret from the provider.
func (providerchef *Providerchef) GetSecret(ctx context.Context, ref v1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return nil, fmt.Errorf("GetSecret yet to implement")
}

// GetSecretMap returns multiple k/v pairs from the provider, for dataFrom.extract.
func (providerchef *Providerchef) GetSecretMap(ctx context.Context, ref v1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, fmt.Errorf("GetSecretMap yet to implement")
}

// ValidateStore checks if the provided store is valid.
func (providerchef *Providerchef) ValidateStore(store v1beta1.GenericStore) error {
	chefProvider, err := getChefProvider(store)
	if err != nil {
		return fmt.Errorf(errChefStore, err)
	}
	// check namespace compared to kind
	if err := utils.ValidateSecretSelector(store, chefProvider.Auth.SecretRef.SecretKey); err != nil {
		return fmt.Errorf(errChefStore, err)
	}
	return nil
}

// getChefProvider validates the incoming store and return the chef provider.
func getChefProvider(store v1beta1.GenericStore) (*v1beta1.ChefProvider, error) {
	if store == nil {
		return nil, fmt.Errorf(errMissingStore)
	}
	storeSpec := store.GetSpec()
	if storeSpec == nil {
		return nil, fmt.Errorf(errMisingStoreSpec)
	}
	provider := storeSpec.Provider
	if provider == nil {
		return nil, fmt.Errorf(errMissingProvider)
	}
	chefProvider := storeSpec.Provider.Chef
	if chefProvider == nil {
		return nil, fmt.Errorf(errMissingChefProvider)
	}
	if chefProvider.Name == "" {
		return chefProvider, fmt.Errorf(errMissingName)
	}
	if chefProvider.BaseURL == "" {
		return chefProvider, fmt.Errorf(errMissingBaseURL)
	}
	// check valid URL
	if _, err := url.ParseRequestURI(chefProvider.BaseURL); err != nil {
		return chefProvider, fmt.Errorf(errChefInvalidURL, err)
	}
	if chefProvider.Auth == nil {
		return chefProvider, fmt.Errorf(errMissingAuth)
	}
	if chefProvider.Auth.SecretRef.SecretKey.Key == "" {
		return chefProvider, fmt.Errorf(errMissingSecretKey)
	}

	return chefProvider, nil
}

/*
type ChefClient interface {
	// DataBagItem fetch single data bag item "item"
	DataBagItem(bag string, item string) (chef.DataBagItem, error)

	// AllDataBagItems get all items in a data bag
	AllDataBagItems(bag string) (map[string]map[string]interface{}, error)
}

// DataBagItem fetch single data bag item "item"
// Return the content as unmarshalled JSON
func (c chefclient) DataBagItem(bag string, item string) (chef.DataBagItem, error) {
	databagitem, err := c.chefClient.DataBags.GetItem(bag, item)
	if err != nil {
		//klog.Infof("Issue getting data bag %s item %s: %s", bag, item, err)
		return nil, err
	}
	return databagitem, nil
}
*/
