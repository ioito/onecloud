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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type SDnsZoneListOptions struct {
		options.BaseListOptions
	}
	R(&SDnsZoneListOptions{}, "dns-zone-list", "List dns zones", func(s *mcclient.ClientSession, opts *SDnsZoneListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.DnsZones.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.DnsZones.GetColumns(s))
		return nil
	})

	type DnsZoneCreateOptions struct {
		NAME      string
		ZONE_TYPE string `choices:"PublicZone|PrivateZone" metavar:"zone_type"`
		Options   string
	}

	R(&DnsZoneCreateOptions{}, "dns-zone-create", "Create dns zone", func(s *mcclient.ClientSession, opts *DnsZoneCreateOptions) error {
		params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
		params.Remove("optsions")
		if len(opts.Options) > 0 {
			options, err := jsonutils.Parse([]byte(opts.Options))
			if err != nil {
				return errors.Wrapf(err, "jsonutils.Parse")
			}
			params.Add(options, "options")
		}
		result, err := modules.DnsZones.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
