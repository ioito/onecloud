// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cloudid

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type SamlProviderListOptions struct {
		options.BaseListOptions
		CloudaccountId string `help:"Cloudaccount Id"`
	}
	R(&SamlProviderListOptions{}, "saml-provider-list", "List saml providers", func(s *mcclient.ClientSession, opts *SamlProviderListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.SamlProviders.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.SamlProviders.GetColumns(s))
		return nil
	})

	type SamlProviderCreateOptions struct {
		NAME              string `help:"Saml Provider name"`
		CLOUDACCOUNT_ID   string `help:"Cloudaccount Id"`
		METADATA_DOCUMENT string `help:"Saml metadata"`
	}

	R(&SamlProviderCreateOptions{}, "saml-provider-create", "Create saml provider", func(s *mcclient.ClientSession, opts *SamlProviderCreateOptions) error {
		result, err := modules.SamlProviders.Create(s, jsonutils.Marshal(opts))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type SamlProviderIdOption struct {
		ID string `help:"Saml Provider Id"`
	}

	R(&SamlProviderIdOption{}, "saml-provider-show", "Show saml provider", func(s *mcclient.ClientSession, opts *SamlProviderIdOption) error {
		result, err := modules.SamlProviders.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&SamlProviderIdOption{}, "saml-provider-delete", "Delete saml provider", func(s *mcclient.ClientSession, opts *SamlProviderIdOption) error {
		result, err := modules.SamlProviders.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
