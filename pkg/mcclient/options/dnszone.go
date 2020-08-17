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

type SDnsZoneListOptions struct {
	BaseListOptions
}

func (opts *SDnsZoneListOptions) Params() (jsonutils.JSONObject, error) {
	return ListStructToParams(opts)
}

type SDnsZoneIdOptions struct {
	ID string `help:"Dns zone Id or Name"`
}

func (opts *SDnsZoneIdOptions) GetId() string {
	return opts.ID
}

func (opts *SDnsZoneIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type DnsZoneCreateOptions struct {
	NAME           string
	ZONE_TYPE      string   `choices:"PublicZone|PrivateZone" metavar:"zone_type"`
	VpcIds         []string `help:"Vpc Ids"`
	CloudaccountId string   `help:"Cloudaccount id"`
	Options        string
}

func (opts *DnsZoneCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(opts.NAME), "name")
	params.Add(jsonutils.NewString(opts.ZONE_TYPE), "zone_type")
	if len(opts.Options) > 0 {
		options, err := jsonutils.Parse([]byte(opts.Options))
		if err != nil {
			return nil, errors.Wrapf(err, "jsonutils.Parse")
		}
		params.Add(options, "options")
	}
	if len(opts.VpcIds) > 0 {
		params.Add(jsonutils.Marshal(opts.VpcIds), "vpc_ids")
	}
	if len(opts.CloudaccountId) > 0 {
		params.Add(jsonutils.NewString(opts.CloudaccountId), "cloudaccount_id")
	}
	return params, nil
}

type DnsZoneCapabilitiesOptions struct {
}

func (opts *DnsZoneCapabilitiesOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type DnsZoneCacheOptions struct {
	SDnsZoneIdOptions
	CLOUDACCOUNT_ID string
}

func (opts *DnsZoneCacheOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"cloudaccount_id": opts.CLOUDACCOUNT_ID}), nil
}

type DnsZoneUncacheOptions struct {
	SDnsZoneIdOptions
	CLOUDACCOUNT_ID string
}

func (opts *DnsZoneUncacheOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"cloudaccount_id": opts.CLOUDACCOUNT_ID}), nil
}

type DnsZoneAddVpcsOptions struct {
	SDnsZoneIdOptions
	VPC_IDS string
}

func (opts *DnsZoneAddVpcsOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"vpc_ids": opts.VPC_IDS}), nil
}

type DnsZoneRemoveVpcsOptions struct {
	SDnsZoneIdOptions
	VPC_IDS string
}

func (opts *DnsZoneRemoveVpcsOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"vpc_ids": opts.VPC_IDS}), nil
}
