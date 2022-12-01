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
package fake

import (
	"fmt"

	"github.com/go-chef/chef"
)

type DataBagService struct {
	ChefMockClient
	MockContent   map[string]interface{}
	MockListItems chef.DataBagListResult
}

type ChefMockClient struct {
	Client chef.Client
}

func NewMockClient() *ChefMockClient {
	return &ChefMockClient{}
}

// GetItems returns []onepassword.Item, you must preload.
func (mockClient *ChefMockClient) ListItems(databagName string) (chef.DataBagListResult, error) {
	// if len(mockClient.MockListItems) > 1 {
	// 	return mockClient.MockListItems, nil
	// }
	ret := make(map[string]string)
	ret["item-01"] = "https://chef.com/organizations/dev/data/" + databagName + "/item-01"
	return ret, nil
	//return chef.DataBagListResult{}, errors.New("status 404: " + databagName + " not found")
}

// GetItem returns a *onepassword.Item, you must preload.
func (mockClient *ChefMockClient) GetItem(dataBagName string, databagItemName string) (chef.DataBagItem, error) {
	// if len(mockClient.MockListItems) > 1 {
	// 	return mockClient.MockContent, nil
	// }
	ret := make(map[string]interface{})
	jsonstring := fmt.Sprintf(`{"id":"%s","some_key":"fe7f29ede349519a1","some_password":"dolphin_123zc","some_username":"testuser"}`, databagItemName)
	ret[databagItemName] = jsonstring
	return ret, nil
	//return nil, errors.New("status 404: " + databagItemName + " not found " + "in " + dataBagName)
}

// AddPredictableVault adds vaults to the mock client in a predictable way.
// func (mockClient *ChefMockClient) AddPredictableListItems(databagName string) *ChefMockClient {
// 	mockClient.MockListItems["item-01"] = "https://chef.com/organizations/dev/data/" + databagName + "/item-01"
// 	// mockClient.MockListItems["item-02"] = "https://chef.com/organizations/dev/data/" + databagName + "/item-02"
// 	return mockClient
// }

// AddPredictableVault adds vaults to the mock client in a predictable way.
// func (mockClient *ChefMockClient) AddPredictableGetItem() *ChefMockClient {
// 	for item, _ := range mockClient.MockListItems {
// 		if item == "item-01" {
// 			jsonstring := fmt.Sprintf(`{"id":"%s","some_key":"fe7f29ede349519a1","some_password":"dolphin_123zc","some_username":"testuser"}`, item)
// 			mockClient.MockContent[item] = jsonstring
// 		}
// 	}
// 	return mockClient
// }
