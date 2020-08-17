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

package compute

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type SDnsTrafficPolicyListOptions struct {
		options.BaseListOptions
	}
	R(&SDnsTrafficPolicyListOptions{}, "dns-trafficpolicy-list", "List dns trafficpolicy", func(s *mcclient.ClientSession, opts *SDnsTrafficPolicyListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.DnsTrafficPolicies.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.DnsTrafficPolicies.GetColumns(s))
		return nil
	})

	type DnsTrafficPolicyCreateOptions struct {
		NAME   string
		Params string
	}

	R(&DnsTrafficPolicyCreateOptions{}, "dns-trafficpolicy-create", "Create dns traffic policy", func(s *mcclient.ClientSession, opts *DnsTrafficPolicyCreateOptions) error {
		params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
		result, err := modules.DnsTrafficPolicies.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
