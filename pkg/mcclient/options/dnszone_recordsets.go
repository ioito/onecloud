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

package options

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SDnsRecordSetListOptions struct {
	BaseListOptions
	DnsZoneId string `help:"DnsZone Id or Name"`
}

func (opts *SDnsRecordSetListOptions) Params() (jsonutils.JSONObject, error) {
	return ListStructToParams(opts)
}

type DnsRecordSetCreateOptions struct {
	NAME        string
	DNS_ZONE_ID string
	Options     string
}

func (opts *DnsRecordSetCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(opts.NAME), "name")
	params.Add(jsonutils.NewString(opts.DNS_ZONE_ID), "dns_zone_id")
	if len(opts.Options) > 0 {
		options, err := jsonutils.Parse([]byte(opts.Options))
		if err != nil {
			return nil, errors.Wrapf(err, "jsonutils.Parse(%s)", opts.Options)
		}
		params.Add(options, "params")
	}
	return params, nil
}

type DnsRecordSetIdOptions struct {
	ID string
}

func (opts *DnsRecordSetIdOptions) GetId() string {
	return opts.ID
}

func (opts *DnsRecordSetIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}
