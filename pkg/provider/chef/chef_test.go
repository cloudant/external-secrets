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
)

const (
	name    = "chef-demo-user"
	baseURL = "https://chef.cloudant.com/organizations/myorg/"
)

type ValidateStoreTestCase struct {
	store *esv1beta1.SecretStore
	err   error
}

type storeModifier func(*esv1beta1.SecretStore) *esv1beta1.SecretStore

func makeSecretStore(name, baseURL string, fn ...storeModifier) *esv1beta1.SecretStore {
	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Chef: &esv1beta1.ChefProvider{
					Name:    name,
					BaseURL: baseURL,
				},
			},
		},
	}

	for _, f := range fn {
		store = f(store)
	}
	return store
}

// minimal TestCases written, more to be added.
func TestValidateStore(t *testing.T) {
	testCases := []ValidateStoreTestCase{
		{
			store: makeSecretStore("", baseURL),
			err:   fmt.Errorf("received invalid Chef SecretStore resource: missing Name"),
		},
		{
			store: makeSecretStore(name, ""),
			err:   fmt.Errorf("received invalid Chef SecretStore resource: missing BaseURL"),
		},
		{
			store: makeSecretStore(name, baseURL),
			err:   fmt.Errorf("received invalid Chef SecretStore resource: cannot initialize Chef Client: no valid authType was specified"),
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
