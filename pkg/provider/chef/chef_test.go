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
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/go-chef/chef"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	fake "github.com/external-secrets/external-secrets/pkg/provider/chef/fake"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	name                     = "chef-demo-user"
	baseURL                  = "https://chef.cloudant.com/organizations/myorg/"
	noEndSlashInvalidBaseURL = "no end slash invalid base URL"
	baseInvalidURL           = "invalid base URL/"
	authName                 = "chef-demo-auth-name"
	authKey                  = "chef-demo-auth-key"
	authNamespace            = "chef-demo-auth-namespace"
	kind                     = "SecretStore"
	apiversion               = "external-secrets.io/v1beta1"
)

type chefTestCase struct {
	mockClient      *fake.ChefMockClient
	databagName     string
	databagItemName string
	property        string
	ref             *esv1beta1.ExternalSecretDataRemoteRef
	apiErr          error
	expectError     string
	expectedData    map[string][]byte
	expectedByte    []byte
}

type ValidateStoreTestCase struct {
	store *esv1beta1.SecretStore
	err   error
}

type storeModifier func(*esv1beta1.SecretStore) *esv1beta1.SecretStore

func makeValidChefTestCase() *chefTestCase {
	smtc := chefTestCase{
		mockClient:      &fake.ChefMockClient{},
		databagName:     "databag01",
		databagItemName: "item01",
		property:        "",
		apiErr:          nil,
		expectError:     "",
		expectedData:    map[string][]byte{"item01": []byte(`"https://chef.com/organizations/dev/data/databag01/item01"`)},
		expectedByte:    []byte(`{"item01":"{\"id\":\"databag01-item01\",\"some_key\":\"fe7f29ede349519a1\",\"some_password\":\"dolphin_123zc\",\"some_username\":\"testuser\"}"}`),
	}

	smtc.ref = makeValidRef(smtc.databagName, smtc.databagItemName, smtc.property)
	smtc.mockClient.WithListItems(smtc.databagName, smtc.apiErr)
	smtc.mockClient.WithItem(smtc.databagName, smtc.databagItemName, smtc.apiErr)
	return &smtc
}

func makeInValidChefTestCase() *chefTestCase {
	smtc := chefTestCase{
		mockClient:      &fake.ChefMockClient{},
		databagName:     "databag01",
		databagItemName: "item03",
		property:        "",
		apiErr:          errors.New("unable to convert databagItem into JSON"),
		expectError:     "unable to convert databagItem into JSON",
		expectedData:    nil,
		expectedByte:    nil,
	}

	smtc.ref = makeValidRef(smtc.databagName, smtc.databagItemName, smtc.property)
	smtc.mockClient.WithListItems(smtc.databagName, smtc.apiErr)
	smtc.mockClient.WithItem(smtc.databagName, smtc.databagItemName, smtc.apiErr)
	return &smtc
}

func makeValidRef(databag, dataitem, property string) *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key:      databag + "/" + dataitem,
		Property: property,
	}
}

func makeinValidRef() *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key: "",
	}
}

func makeValidChefTestCaseCustom(tweaks ...func(smtc *chefTestCase)) *chefTestCase {
	smtc := makeValidChefTestCase()
	for _, fn := range tweaks {
		fn(smtc)
	}
	return smtc
}

func TestChefGetSecret(t *testing.T) {
	nilClient := func(smtc *chefTestCase) {
		smtc.mockClient = nil
		smtc.expectedByte = nil
		smtc.apiErr = errors.New("provider chef is not initialized")
		smtc.expectError = "provider chef is not initialized"
	}

	invalidDatabagName := func(smtc *chefTestCase) {
		smtc.apiErr = errors.New("invalid format. Expected value 'databagName/databagItemName")
		smtc.databagName = "databag02"
		smtc.expectedByte = nil
		smtc.ref = makeinValidRef()
		smtc.expectError = "invalid format. Expected value 'databagName/databagItemName"
	}

	invalidDatabagItemName := func(smtc *chefTestCase) {
		smtc.apiErr = errors.New("no Databag Item found")
		smtc.expectError = "no Databag Item found"
		smtc.databagName = "databag01"
		smtc.databagItemName = "item02"
		smtc.expectedByte = nil
		smtc.ref = makeValidRef(smtc.databagName, smtc.databagItemName, "")
	}

	noProperty := func(smtc *chefTestCase) {
		smtc.apiErr = errors.New("property is not found in Databag item")
		smtc.expectError = "property is not found in Databag item"
		smtc.databagName = "databag01"
		smtc.databagItemName = "item01"
		smtc.expectedByte = nil
		smtc.ref = makeValidRef(smtc.databagName, smtc.databagItemName, "findProperty")
	}

	withProperty := func(smtc *chefTestCase) {
		smtc.expectedByte = []byte("foundProperty")
		smtc.apiErr = nil
		smtc.databagName = "databag03"
		smtc.databagItemName = "item03"
		smtc.ref = makeValidRef(smtc.databagName, smtc.databagItemName, "findProperty")
	}

	successCases := []*chefTestCase{
		makeValidChefTestCase(),
		makeValidChefTestCaseCustom(nilClient),
		makeValidChefTestCaseCustom(invalidDatabagName),
		makeValidChefTestCaseCustom(invalidDatabagItemName),
		makeValidChefTestCaseCustom(noProperty),
		makeValidChefTestCaseCustom(withProperty),
		makeInValidChefTestCase(),
	}

	sm := Providerchef{
		databagService: &chef.DataBagService{},
	}
	for k, v := range successCases {
		sm.databagService = v.mockClient
		out, err := sm.GetSecret(context.Background(), *v.ref)
		if err != nil && !utils.ErrorContains(err, v.expectError) {
			t.Errorf("test failed! want: %v, got: %v", v.expectError, err)
		}
		if string(out) != string(v.expectedByte) {
			t.Errorf("[%d] unexpected secret: expected %s, got %s", k, v.expectedByte, string(out))
		}
	}
}

func TestChefGetSecretMap(t *testing.T) {
	nilClient := func(smtc *chefTestCase) {
		smtc.mockClient = nil
		smtc.expectedByte = nil
		smtc.apiErr = errors.New("provider chef is not initialized")
	}

	invalidDatabagName := func(smtc *chefTestCase) {
		smtc.expectedByte = nil
		smtc.apiErr = errors.New("could not get secret data from provider")
		smtc.databagName = "databag02"
		smtc.expectedByte = nil
		smtc.ref = makeinValidRef()
	}

	successCases := []*chefTestCase{
		makeValidChefTestCaseCustom(nilClient),
		makeValidChefTestCaseCustom(invalidDatabagName),
	}

	pc := Providerchef{
		databagService: &chef.DataBagService{},
	}
	for k, v := range successCases {
		pc.databagService = v.mockClient
		out, err := pc.GetSecretMap(context.Background(), *v.ref)
		if err != nil && utils.ErrorContains(err, v.expectError) {
			t.Errorf("test failed! want: %v, got: %v", v.apiErr, err)
		}
		if string(out["item01"]) != string(v.expectedByte) {
			t.Errorf("[%d] unexpected secret: expected %s, got %s", k, v.expectedByte, out)
		}
	}
}

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
			err:   fmt.Errorf("received invalid Chef SecretStore resource: unable to parse URL: parse \"invalid base URL/\": invalid URI for request"),
		},
		{
			store: makeSecretStore(name, noEndSlashInvalidBaseURL, makeAuth(authName, authNamespace, authKey)),
			err:   fmt.Errorf("received invalid Chef SecretStore resource: server URL does not end with slash(/)"),
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
			t.Errorf("test failed! want: %v, got: %v", tc.err, err)
		} else if tc.err == nil && err != nil {
			t.Errorf("want: nil got: err %v", err)
		} else if tc.err != nil && err == nil {
			t.Errorf("want: err %v got: nil", tc.err)
		}
	}
}

func TestNewClient(t *testing.T) {
	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Chef: &esv1beta1.ChefProvider{
					Auth:      makeAuth(authName, authNamespace, authKey),
					UserName:  name,
					ServerURL: baseURL,
				},
			},
		},
	}

	expected := fmt.Sprintf("could not fetch SecretKey Secret: secrets %q not found", authName)
	ctx := context.TODO()

	kube := clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "creds",
			Namespace: "default",
		}, TypeMeta: metav1.TypeMeta{
			Kind:       kind,
			APIVersion: apiversion,
		},
	}).Build()
	pc := Providerchef{databagService: &fake.ChefMockClient{}}
	_, err := pc.NewClient(ctx, store, kube, "default")
	if !ErrorContains(err, expected) {
		t.Errorf("CheckNewClient unexpected error: %s, expected: '%s'", err.Error(), expected)
	}
}

func ErrorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}

func TestValidate(t *testing.T) {
	pc := Providerchef{}
	var mockClient *fake.ChefMockClient
	pc.userService = mockClient
	pc.clientName = "correctUser"
	_, err := pc.Validate()
	t.Log("Error: ", err)
	pc.clientName = "wrongUser"
	_, err = pc.Validate()
	t.Log("Error: ", err)
}
