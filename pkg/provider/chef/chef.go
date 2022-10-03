package chef

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/go-chef/chef"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errChefStore                             = "received invalid Chef SecretStore resource: %w"
	errUnexpectedStoreSpec                   = "unexpected store spec"
	errMissingName                           = "missing Name"
	errMissingBaseURL                        = "missing BaseURL"
	errMissingAuth                           = "cannot initialize Chef Client: no valid authType was specified"
	errMissingPublicKey                      = "missing Public Key"
	errInvalidClusterStoreMissingPKNamespace = "invalid ClusterSecretStore: missing Chef PublicKey Namespace"
	errFetchK8sSecret                        = "could not fetch PublicKey Secret: %w"
	errChefInvalidURL                        = "unable to parse URL: %w"
	errChefInvalidName                       = "invalid name: allowed values are lowecase letters, numbers, hyphens and underscores"
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
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider.Chef == nil {
		return nil, errors.New(errUnexpectedStoreSpec)
	}
	chefSpec := storeSpec.Provider.Chef
	if chefSpec.Name == "" {
		return nil, fmt.Errorf(errMissingName)
	}
	if chefSpec.BaseURL == "" {
		return nil, fmt.Errorf(errMissingBaseURL)
	}
	if chefSpec.Auth == nil {
		return nil, fmt.Errorf(errMissingAuth)
	}
	if len(chefSpec.Auth.SecretRef.PublicKey.Key) == 0 || &chefSpec.Auth.SecretRef.PublicKey.Key == nil {
		return nil, fmt.Errorf(errMissingPublicKey)
	}
	credentialsSecret := &corev1.Secret{}
	objectKey := types.NamespacedName{
		Name:      chefSpec.Auth.SecretRef.PublicKey.Name,
		Namespace: namespace,
	}
	if store.GetObjectKind().GroupVersionKind().Kind == v1beta1.ClusterSecretStoreKind {
		objectKey.Namespace = *chefSpec.Auth.SecretRef.PublicKey.Namespace
	} else {
		return nil, fmt.Errorf(errInvalidClusterStoreMissingPKNamespace)
	}

	err := kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return nil, fmt.Errorf(errFetchK8sSecret, err)
	}

	publickey := credentialsSecret.Data[chefSpec.Auth.SecretRef.PublicKey.Key]
	if (publickey == nil) || (len(publickey) == 0) {
		return nil, fmt.Errorf(errMissingPublicKey)
	}

	client, err := chef.NewClient(&chef.Config{
		Name:    chefSpec.Name,
		Key:     string(publickey),
		BaseURL: chefSpec.BaseURL,
	})

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
func (provider *Providerchef) ValidateStore(store v1beta1.GenericStore) error {
	storeSpec := store.GetSpec()
	if storeSpec == nil {
		return fmt.Errorf(errChefStore, fmt.Errorf(errUnexpectedStoreSpec))
	}
	if storeSpec.Provider == nil {
		return fmt.Errorf(errChefStore, fmt.Errorf(errUnexpectedStoreSpec))
	}
	chefSpec := storeSpec.Provider.Chef
	if chefSpec == nil {
		return fmt.Errorf(errChefStore, fmt.Errorf(errUnexpectedStoreSpec))
	}
	if chefSpec.BaseURL == "" {
		return fmt.Errorf(errChefStore, fmt.Errorf(errMissingBaseURL))
	}
	//check valid URL
	if _, err := url.Parse(chefSpec.BaseURL); err != nil {
		return fmt.Errorf(errChefStore, fmt.Errorf(errChefInvalidURL, err))
	}
	if chefSpec.Name == "" {
		return fmt.Errorf(errChefStore, fmt.Errorf(errMissingName))
	}
	//check if Name contains only lowecase letters, numbers, hyphens and underscores
	var validNameRegEx = regexp.MustCompile(`^[a-z0-9\_\-]*$`)
	if !validNameRegEx.MatchString(chefSpec.Name) {
		return fmt.Errorf(errChefStore, fmt.Errorf(errChefInvalidName))
	}
	if chefSpec.Auth == nil {
		return fmt.Errorf(errChefStore, fmt.Errorf(errMissingAuth))
	}
	if chefSpec.Auth.SecretRef.PublicKey.Key == "" {
		return fmt.Errorf(errChefStore, fmt.Errorf(errMissingPublicKey))
	}
	// check namespace compared to kind
	if err := utils.ValidateSecretSelector(store, chefSpec.Auth.SecretRef.PublicKey); err != nil {
		return fmt.Errorf(errChefStore, err)
	}
	return nil
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
