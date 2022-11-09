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
	"fmt"
	"testing"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	name           = "chef-demo-user"
	baseURL        = "https://chef.cloudant.com/organizations/myorg/"
	baseInvalidURL = "invalid base URL"
	authName       = "chef-demo-auth-name"
	authKey        = "chef-demo-auth-key"
	authNamespace  = "chef-demo-auth-namespace"
)

type ValidateStoreTestCase struct {
	store *esv1beta1.SecretStore
	err   error
}

type storeModifier func(*esv1beta1.SecretStore) *esv1beta1.SecretStore

func makeSecretStore(name, baseURL string, auth *esv1beta1.ChefAuth, fn ...storeModifier) *esv1beta1.SecretStore {
	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Chef: &esv1beta1.ChefProvider{
					UserName:  name,
					ServerURL: baseURL,
					Auth:      auth,
				},
			},
		},
	}

	for _, f := range fn {
		store = f(store)
	}
	return store
}

func makeAuth(name, namespace, key string) *esv1beta1.ChefAuth {
	return &esv1beta1.ChefAuth{
		SecretRef: esv1beta1.ChefAuthSecretRef{
			SecretKey: v1.SecretKeySelector{
				Name:      name,
				Key:       key,
				Namespace: &namespace,
			},
		},
	}
}

// minimal TestCases written, more to be added.
func TestValidateStore(t *testing.T) {
	testCases := []ValidateStoreTestCase{
		{
			store: makeSecretStore("", baseURL, makeAuth(authName, authNamespace, authKey)),
			err:   fmt.Errorf("received invalid Chef SecretStore resource: missing Name"),
		},
		{
			store: makeSecretStore(name, "", makeAuth(authName, authNamespace, authKey)),
			err:   fmt.Errorf("received invalid Chef SecretStore resource: missing BaseURL"),
		},
		{
			store: makeSecretStore(name, baseURL, nil),
			err:   fmt.Errorf("received invalid Chef SecretStore resource: cannot initialize Chef Client: no valid authType was specified"),
		},
		{
			store: makeSecretStore(name, baseInvalidURL, makeAuth(authName, authNamespace, authKey)),
			err:   fmt.Errorf("received invalid Chef SecretStore resource: unable to parse URL: parse \"invalid base URL\": invalid URI for request"),
		},
		{
			store: makeSecretStore(name, baseURL, makeAuth(authName, authNamespace, "")),
			err:   fmt.Errorf("received invalid Chef SecretStore resource: missing Secret Key"),
		},
		{
			store: makeSecretStore(name, baseURL, makeAuth(authName, authNamespace, authKey)),
			err:   fmt.Errorf("received invalid Chef SecretStore resource: namespace not allowed with namespaced SecretStore"),
		},
		{
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: nil,
				},
			},
			err: fmt.Errorf("received invalid Chef SecretStore resource: missing provider"),
		},
		{
			store: &esv1beta1.SecretStore{
				Spec: esv1beta1.SecretStoreSpec{
					Provider: &esv1beta1.SecretStoreProvider{
						Chef: nil,
					},
				},
			},
			err: fmt.Errorf("received invalid Chef SecretStore resource: missing chef provider"),
		},
	}
	pc := Providerchef{}
	for _, tc := range testCases {
		err := pc.ValidateStore(tc.store)
		if tc.err != nil && err != nil && err.Error() != tc.err.Error() {
			t.Errorf("test failed! want %v, got %v", tc.err, err)
		} else if tc.err == nil && err != nil {
			t.Errorf("want nil got err %v", err)
		} else if tc.err != nil && err == nil {
			t.Errorf("want err %v got nil", tc.err)
		}
	}
}
