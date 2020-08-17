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
	"fmt"

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
	db.SExternalizedResourceBaseManager

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
	db.SExternalizedResourceBase
	SDnsZoneResourceBase

	DnsType             string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	DnsValue            string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	TTL                 int    `nullable:"false" list:"user" json:"ttl"`
	DnsTrafficePolicyId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
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

func (self *SDnsRecordSet) GetDnsTrafficPolicies(policyType string) ([]SDnsTrafficPolicy, error) {
	q := DnsTrafficPolicyManager.Query()
	if len(policyType) > 0 {
		q = q.Equals("policy_type", policyType)
	}
	policies := []SDnsTrafficPolicy{}
	err := db.FetchModelObjects(DnsTrafficPolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return policies, nil
}

func (self *SDnsRecordSet) GetDnsZone() (*SDnsZone, error) {
	dnsZone, err := DnsZoneManager.FetchById(self.DnsZoneId)
	if err != nil {
		return nil, errors.Wrapf(err, "DnsZoneManager.FetchById(%s)", self.DnsZoneId)
	}
	return dnsZone.(*SDnsZone), nil
}

func (self *SDnsRecordSet) newFromDnsTrafficPolicy(ctx context.Context, userCred mcclient.TokenCredential, policyType string, params *jsonutils.JSONDict) (*SDnsTrafficPolicy, error) {
	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return nil, errors.Wrapf(err, "GetDnsZone")
	}

	policy := &SDnsTrafficPolicy{}
	policy.SetModelManager(DnsTrafficPolicyManager, policy)
	name := fmt.Sprintf("traffic policy for %s", self.Name)
	policy.Name, err = db.GenerateName(DnsTrafficPolicyManager, userCred, name)
	if err != nil {
		return nil, errors.Wrapf(err, "db.GenerateName")
	}
	policy.PolicyType = policyType
	policy.Params = params
	err = DnsTrafficPolicyManager.TableSpec().Insert(ctx, policy)
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}
	SyncCloudDomain(userCred, policy, dnsZone.GetOwnerId())
	return policy, nil
}

func (self *SDnsRecordSet) SyncDnsTrafficPolicy(ctx context.Context, userCred mcclient.TokenCredential, policy cloudprovider.ICloudDnsTrafficPolicy) error {
	policyType, params := policy.GetPolicyType(), policy.GetParams()
	policies, err := self.GetDnsTrafficPolicies(string(policyType))
	if err != nil {
		return errors.Wrapf(err, "GetDnsTrafficPolicies(%s)", policyType)
	}
	policyId := ""
	for i := range policies {
		if policies[i].Params == params || (params != nil && policies[i].Params != nil && params.Equals(policies[i].Params)) {
			policyId = policies[i].Id
			break
		}
	}
	if len(policyId) == 0 {
		policy, err := self.newFromDnsTrafficPolicy(ctx, userCred, string(policyType), params)
		if err != nil {
			return errors.Wrapf(err, "newFromDnsTrafficPolicy")
		}
		policyId = policy.Id
	}
	_, err = db.Update(self, func() error {
		self.DnsTrafficePolicyId = policyId
		return nil
	})
	return err
}
