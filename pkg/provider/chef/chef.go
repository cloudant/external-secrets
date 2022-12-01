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
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-chef/chef"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
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
	errChefProvider                          = "missing or invalid spec: %w"
	errUninitalizedChefProvider              = "provider chef is not initialized"
	errNoDatabagsFound                       = "no Databags found"
	errNoDatabagItemFound                    = "no Databag Item found"
	errNoDatabagItemContentFound             = "no Databag Item's content found"
	errNoDatabagItemPropertyFound            = "property is not found in Databag item"
	errUnableToConvertToJSON                 = "unable to convert databagItem into JSON"
	errInvalidFormat                         = "invalid format. Expected value 'databagName/databagItemName'"
	errStoreValidateFailed                   = "unable to validate provided store. Check if username, serverUrl and privateKey are correct"
	errServerURLNoEndSlash                   = "server URL does not end with slash(/)"
)

type ChefInterface interface {
	GetItem(string, string) (chef.DataBagItem, error)
	ListItems(string) (*chef.DataBagListResult, error)
}

type Providerchef struct {
	//chefClient *chef.Client
	ChefInterface chef.DataBagService
}

// https://github.com/external-secrets/external-secrets/issues/644
var _ v1beta1.SecretsClient = &Providerchef{}
var _ v1beta1.Provider = &Providerchef{}

var log = ctrl.Log.WithName("provider").WithName("chef").WithName("secretsmanager")

func init() {
	v1beta1.Register(&Providerchef{}, &v1beta1.SecretStoreProvider{
		Chef: &v1beta1.ChefProvider{},
	})
}

func (providerchef *Providerchef) NewClient(ctx context.Context, store v1beta1.GenericStore, kube kclient.Client, namespace string) (v1beta1.SecretsClient, error) {
	// handle validation of clustersecretstore, serstore, externalserstore

	chefProvider, err := getChefProvider(store)
	if err != nil {
		return nil, fmt.Errorf(errChefProvider, err)
	}
	credentialsSecret := &corev1.Secret{}
	objectKey := types.NamespacedName{
		Name:      chefProvider.Auth.SecretRef.SecretKey.Name,
		Namespace: namespace,
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
		Name:    chefProvider.UserName,
		Key:     string(secretKey),
		BaseURL: chefProvider.ServerURL,
	})
	if err != nil {
		return nil, fmt.Errorf(errChefClient, err)
	}
	providerchef.ChefInterface = *client.DataBags
	return providerchef, nil
}

// Close closes the client connection.
func (providerchef *Providerchef) Close(ctx context.Context) error {
	return nil
}

// Validate checks if the client is configured correctly
// to be able to retrieve secrets from the provider.
func (providerchef *Providerchef) Validate() (v1beta1.ValidationResult, error) {
	// serverURL := providerchef.chefClient.BaseURL.String()
	// endsWithSlash := strings.HasSuffix(serverURL, "/")
	// if !endsWithSlash {
	// 	return v1beta1.ValidationResultError, fmt.Errorf(errServerURLNoEndSlash)
	// }

	// _, err := providerchef.chefClient.Users.Get(providerchef.chefClient.Auth.ClientName)
	// if err != nil {
	// 	return v1beta1.ValidationResultError, fmt.Errorf(errStoreValidateFailed)
	// }
	return v1beta1.ValidationResultReady, nil
}

// GetAllSecrets Retrieves a map[string][]byte with the Databag names as key and the Databag's Items as secrets.
// Retrives all DatabagItems of a Databag.
func (providerchef *Providerchef) GetAllSecrets(ctx context.Context, ref v1beta1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, fmt.Errorf("dataFrom.find not suppported")
}

// GetSecret returns a databagItem present in the databag. format example: databagName/databagItemName.
func (providerchef *Providerchef) GetSecret(ctx context.Context, ref v1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if utils.IsNil(providerchef.ChefInterface) {
		return nil, fmt.Errorf(errUninitalizedChefProvider)
	}
	fmt.Println(ref.Key, ref.Property)

	key := ref.Key
	databagName := ""
	databagItem := ""
	nameSplitted := strings.Split(key, "/")
	if len(nameSplitted) > 1 {
		databagName = nameSplitted[0]
		databagItem = nameSplitted[1]
	}
	log.Info("fetching secret value", "databag Name:", databagName, "databag Item:", databagItem)
	if len(databagName) != 0 && len(databagItem) != 0 {
		return getSingleDatabagItem(providerchef, databagName, databagItem, ref.Property)
	}

	return nil, fmt.Errorf(errInvalidFormat)
}

func getSingleDatabagItem(providerchef *Providerchef, dataBagName, databagItemName, propertyName string) ([]byte, error) {
	ditem, err := providerchef.ChefInterface.GetItem(dataBagName, databagItemName)
	if err != nil {
		return nil, fmt.Errorf(errNoDatabagItemFound)
	}

	jsonByte, err := json.Marshal(ditem)
	if err != nil {
		return nil, fmt.Errorf(errUnableToConvertToJSON)
	}

	if propertyName != "" {
		return getPropertyFromDatabagItem(string(jsonByte), propertyName)
	}

	return jsonByte, nil
}

func getPropertyFromDatabagItem(jsonString, propertyName string) ([]byte, error) {
	result := gjson.Get(jsonString, propertyName)

	if !result.Exists() {
		return nil, fmt.Errorf(errNoDatabagItemPropertyFound)
	}
	return []byte(result.String()), nil
}

// GetSecretMap returns multiple k/v pairs from the provider, for dataFrom.extract.
func (providerchef *Providerchef) GetSecretMap(ctx context.Context, ref v1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	if utils.IsNil(providerchef.ChefInterface) {
		return nil, fmt.Errorf(errUninitalizedChefProvider)
	}
	databagName := ref.Key
	getAllSecrets := make(map[string][]byte)
	log.Info("fetching all items from databag:", databagName)
	dataItems, err := providerchef.ChefInterface.ListItems(databagName)
	if err != nil {
		return nil, fmt.Errorf(errNoDatabagItemFound)
	}

	for dataItem := range *dataItems {
		dItem, err := getSingleDatabagItem(providerchef, databagName, dataItem, "")
		if err != nil {
			fmt.Println(err)
		}
		getAllSecrets[dataItem] = dItem
	}
	return getAllSecrets, nil
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
	if chefProvider.UserName == "" {
		return chefProvider, fmt.Errorf(errMissingName)
	}
	if chefProvider.ServerURL == "" {
		return chefProvider, fmt.Errorf(errMissingBaseURL)
	}
	// check valid URL
	if _, err := url.ParseRequestURI(chefProvider.ServerURL); err != nil {
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
