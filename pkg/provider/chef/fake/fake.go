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
	"errors"
	"fmt"

	"github.com/go-chef/chef"
)

type ChefMockClient struct {
	getItem   func(databagName string, databagItem string) (item chef.DataBagItem, err error)
	listItems func(name string) (data *chef.DataBagListResult, err error)
}

func (mc *ChefMockClient) GetItem(databagName, databagItem string) (item chef.DataBagItem, err error) {
	return mc.getItem(databagName, databagItem)
}

func (mc *ChefMockClient) ListItems(name string) (data *chef.DataBagListResult, err error) {
	return mc.listItems(name)
}

func (mc *ChefMockClient) WithItem(dataBagName, databagItemName string, err error) {
	if mc != nil {
		mc.getItem = func(dataBagName, databagItemName string) (item chef.DataBagItem, err error) {
			ret := make(map[string]interface{})
			if dataBagName == "databag01" && databagItemName == "item01" {
				jsonstring := fmt.Sprintf(`{"id":"%s","some_key":"fe7f29ede349519a1","some_password":"dolphin_123zc","some_username":"testuser"}`, dataBagName+"-"+databagItemName)
				ret[databagItemName] = jsonstring
			} else {
				str := "https://chef.com/organizations/dev/data/" + dataBagName + "/" + databagItemName + ": 404"
				return nil, errors.New(str)
			}

			return ret, nil
		}
	}
}

func (mc *ChefMockClient) WithListItems(databagName string, err error) {
	if mc != nil {
		mc.listItems = func(databagName string) (data *chef.DataBagListResult, err error) {
			ret := make(chef.DataBagListResult)
			jsonstring := fmt.Sprintf("https://chef.com/organizations/dev/data/%s/item01", databagName)
			ret["item01"] = jsonstring
			return &ret, nil
		}
	}
}
