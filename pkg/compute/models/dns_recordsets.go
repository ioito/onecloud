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

package models

import (
	"context"
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SDnsRecordSetManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager

	SDnsZoneResourceBaseManager
}

var DnsRecordSetManager *SDnsRecordSetManager

func init() {
	DnsRecordSetManager = &SDnsRecordSetManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SDnsRecordSet{},
			"dns_recordsets_tbl",
			"dns_recordset",
			"dns_recordsets",
		),
	}
	DnsRecordSetManager.SetVirtualObject(DnsRecordSetManager)

}

type SDnsRecordSet struct {
	db.SEnabledStatusStandaloneResourceBase
	SDnsZoneResourceBase

	DnsType  string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	DnsValue string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	TTL      int64  `nullable:"false" list:"user" json:"ttl"`
}

// 创建
func (manager *SDnsRecordSetManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.DnsRecordSetCreateInput) (api.DnsRecordSetCreateInput, error) {
	return input, httperrors.NewNotImplementedError("Not Implement")
}

// DNS记录列表
func (manager *SDnsRecordSetManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DnsRecordSetListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

// 解析详情
func (self *SDnsRecordSet) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.DnsRecordSetDetails, error) {
	return api.DnsRecordSetDetails{}, nil
}

func (manager *SDnsRecordSetManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.DnsRecordSetDetails {
	rows := make([]api.DnsRecordSetDetails, len(objs))
	enRows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.DnsRecordSetDetails{
			EnabledStatusStandaloneResourceDetails: enRows[i],
		}
	}
	return rows
}

func (self *SDnsRecordSet) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.Delete(ctx, userCred)
}

func (self *SDnsRecordSet) GetDnsTrafficPolicies() ([]SDnsTrafficPolicy, error) {
	sq := DnsRecordSetTrafficPolicyManager.Query("dns_trafficpolicy_id").Equals("dns_recordset_id", self.Id)
	q := DnsTrafficPolicyManager.Query().In("id", sq.SubQuery())
	policies := []SDnsTrafficPolicy{}
	err := db.FetchModelObjects(DnsTrafficPolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return policies, nil
}

func (self *SDnsRecordSet) GetDnsTrafficPolicy(provider string) (*SDnsTrafficPolicy, error) {
	sq := DnsRecordSetTrafficPolicyManager.Query("dns_trafficpolicy_id").Equals("dns_recordset_id", self.Id)
	q := DnsTrafficPolicyManager.Query().In("id", sq.SubQuery()).Equals("provider", provider)
	policies := []SDnsTrafficPolicy{}
	err := db.FetchModelObjects(DnsTrafficPolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	if len(policies) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(policies) > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	}
	return &policies[0], nil
}

func (self *SDnsRecordSet) GetDefaultDnsTrafficPolicy(provider string) (cloudprovider.TDnsPolicyType, cloudprovider.TDnsPolicyTypeValue, error) {
	policy, err := self.GetDnsTrafficPolicy(provider)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return cloudprovider.DnsPolicyTypeSimple, nil, errors.Wrapf(err, "GetDnsTrafficPolicy(%s)", provider)
	}
	if policy != nil {
		if policy.Params != nil {
			return cloudprovider.TDnsPolicyType(policy.PolicyType), cloudprovider.TDnsPolicyTypeValue(policy.Params), nil
		}
		return cloudprovider.TDnsPolicyType(policy.PolicyType), nil, nil
	}
	return cloudprovider.DnsPolicyTypeSimple, nil, nil
}

func (self *SDnsRecordSet) GetDnsZone() (*SDnsZone, error) {
	dnsZone, err := DnsZoneManager.FetchById(self.DnsZoneId)
	if err != nil {
		return nil, errors.Wrapf(err, "DnsZoneManager.FetchById(%s)", self.DnsZoneId)
	}
	return dnsZone.(*SDnsZone), nil
}
