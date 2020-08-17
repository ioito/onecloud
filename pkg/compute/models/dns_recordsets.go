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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
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

	DnsType  string `width:"36" charset:"ascii" nullable:"false" list:"user" update:"domain" create:"domain_required"`
	DnsValue string `width:"36" charset:"ascii" nullable:"false" list:"user" update:"domain" create:"domain_required"`
	TTL      int64  `nullable:"false" list:"user" update:"domain" create:"domain_required" json:"ttl"`
}

// 创建
func (manager *SDnsRecordSetManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.DnsRecordSetCreateInput) (api.DnsRecordSetCreateInput, error) {
	if len(input.DnsZoneId) == 0 {
		return input, httperrors.NewMissingParameterError("dns_zone_id")
	}
	_dnsZone, err := DnsZoneManager.FetchByIdOrName(userCred, input.DnsZoneId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return input, httperrors.NewResourceNotFoundError2("dns_zone", input.DnsZoneId)
		}
		return input, httperrors.NewGeneralError(err)
	}
	dnsZone := _dnsZone.(*SDnsZone)
	for _, policy := range input.TrafficPolicies {
		if len(policy.Provider) == 0 {
			return input, httperrors.NewGeneralError(fmt.Errorf("missing traffic policy provider"))
		}
		factory, err := cloudprovider.GetProviderFactory(policy.Provider)
		if err != nil {
			return input, httperrors.NewGeneralError(errors.Wrapf(err, "invalid provider %s for traffic policy", policy.Provider))
		}
		_dnsTypes := factory.GetSupportedDnsTypes()
		dnsTypes, _ := _dnsTypes[cloudprovider.TDnsZoneType(dnsZone.ZoneType)]
		if ok, _ := utils.InArray(cloudprovider.TDnsType(input.DnsType), dnsTypes); !ok {
			return input, httperrors.NewNotSupportedError("%s %s not supported dns type %s", policy.Provider, dnsZone.ZoneType, input.DnsType)
		}
		_policyTypes := factory.GetSupportedDnsPolicyTypes()
		policyTypes, _ := _policyTypes[cloudprovider.TDnsZoneType(dnsZone.ZoneType)]
		if ok, _ := utils.InArray(cloudprovider.TDnsPolicyType(policy.PolicyType), policyTypes); !ok {
			return input, httperrors.NewNotSupportedError("%s %s not supported policy type %s", policy.Provider, dnsZone.ZoneType, policy.PolicyType)
		}
		if policy.PolicyParams != nil {
		}
		_policyValues := factory.GetSupportedDnsPolicyTypeValues()
		policyValues, _ := _policyValues[cloudprovider.TDnsPolicyType(policy.PolicyType)]
		if len(policyValues) > 0 {
			if policy.PolicyParams == nil {
				return input, httperrors.NewMissingParameterError(fmt.Sprintf("missing %s policy params", policy.Provider))
			}
			if ok, _ := utils.InArray(policy.PolicyParams, policyValues); !ok {
				return input, httperrors.NewNotSupportedError("%s %s %s not support %s", policy.Provider, dnsZone.ZoneType, policy.PolicyType, policy.PolicyParams)
			}
		}
	}
	input.DnsZoneId = dnsZone.Id
	return input, nil
}

func (self *SDnsRecordSet) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	input := api.DnsRecordSetCreateInput{}
	data.Unmarshal(&input)
	for _, policy := range input.TrafficPolicies {
		self.setTrafficPolicy(ctx, userCred, policy.Provider, cloudprovider.TDnsPolicyType(policy.PolicyType), cloudprovider.TDnsPolicyTypeValue(policy.PolicyParams))
	}

	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return
	}
	logclient.AddSimpleActionLog(dnsZone, logclient.ACT_ALLOCATE, data, userCred, true)
	dnsZone.DoSyncRecords(ctx, userCred)
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
	q, err = manager.SDnsZoneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.DnsZoneFilterListBase)
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

func (self *SDnsRecordSet) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	policies, err := self.GetDnsTrafficPolicies()
	if err != nil {
		return errors.Wrapf(err, "GetDnsTrafficPolicies for record %s(%s)", self.Name, self.Id)
	}
	for i := range policies {
		err = self.RemovePolicy(ctx, userCred, policies[i].Id)
		if err != nil {
			return errors.Wrapf(err, "RemovePolicy(%s)", policies[i].Id)
		}
	}
	return self.SEnabledStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (manager *SDnsRecordSetManager) FetchParentId(ctx context.Context, data jsonutils.JSONObject) string {
	parentId, _ := data.GetString("dns_zone_id")
	return parentId
}

func (manager *SDnsRecordSetManager) FilterByParentId(q *sqlchemy.SQuery, parentId string) *sqlchemy.SQuery {
	if len(parentId) > 0 {
		q = q.Equals("dns_zone_id", parentId)
	}
	return q
}

func (manager *SDnsRecordSetManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	parentId := manager.FetchParentId(ctx, data)
	if len(parentId) > 0 {
		dnsZone, err := db.FetchById(DnsZoneManager, parentId)
		if err != nil {
			return nil, errors.Wrapf(err, "db.FetchById(DnsZoneManager, %s)", parentId)
		}
		return dnsZone.(*SDnsZone).GetOwnerId(), nil
	}
	return db.FetchDomainInfo(ctx, data)
}

func (manager *SDnsRecordSetManager) FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	sq := DnsZoneManager.Query("id")
	sq = db.SharableManagerFilterByOwner(DnsZoneManager, sq, userCred, scope)
	return q.In("dns_zone_id", sq.SubQuery())
}

func (manager *SDnsRecordSetManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeDomain
}

func (self *SDnsRecordSet) GetOwnerId() mcclient.IIdentityProvider {
	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return nil
	}
	return dnsZone.GetOwnerId()
}

func (self *SDnsRecordSet) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	policies, err := self.GetDnsTrafficPolicies()
	if err != nil {
		return errors.Wrapf(err, "GetDnsTrafficPolicies")
	}
	for i := range policies {
		err = self.RemovePolicy(ctx, userCred, policies[i].Id)
		if err != nil {
			return errors.Wrapf(err, "RemovePolicy")
		}
	}
	return self.Delete(ctx, userCred)
}

func (self *SDnsRecordSet) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	self.SEnabledStatusStandaloneResourceBase.PreDelete(ctx, userCred)

	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return
	}
	logclient.AddSimpleActionLog(dnsZone, logclient.ACT_ALLOCATE, self, userCred, true)
	dnsZone.DoSyncRecords(ctx, userCred)
}

func (self *SDnsRecordSet) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)

	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return
	}
	logclient.AddSimpleActionLog(self, logclient.ACT_UPDATE, data, userCred, true)
	dnsZone.DoSyncRecords(ctx, userCred)
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

func (self *SDnsRecordSet) syncWithCloudDnsRecord(ctx context.Context, userCred mcclient.TokenCredential, provider string, ext cloudprovider.DnsRecordSet) error {
	_, err := db.Update(self, func() error {
		self.Name = ext.DnsName
		self.Status = ext.Status
		self.TTL = ext.Ttl
		self.DnsType = string(ext.DnsType)
		self.DnsValue = ext.DnsValue
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "update")
	}
	return self.setTrafficPolicy(ctx, userCred, provider, ext.PolicyType, ext.PolicyParams)
}

func (self *SDnsRecordSet) RemovePolicy(ctx context.Context, userCred mcclient.TokenCredential, policyId string) error {
	q := DnsRecordSetTrafficPolicyManager.Query().Equals("dns_recordset_id", self.Id).Equals("dns_trafficpolicy_id", policyId)
	removed := []SDnsRecordSetTrafficPolicy{}
	err := db.FetchModelObjects(DnsRecordSetTrafficPolicyManager, q, &removed)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range removed {
		err = removed[i].Detach(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "Detach")
		}
	}
	return nil
}

func (self *SDnsRecordSet) setTrafficPolicy(ctx context.Context, userCred mcclient.TokenCredential, provider string, policyType cloudprovider.TDnsPolicyType, policyValue cloudprovider.TDnsPolicyTypeValue) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	policy, err := self.GetDnsTrafficPolicy(provider)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return errors.Wrapf(err, "GetDnsTrafficPolicy(%s)", provider)
	}
	if policy != nil {
		if cloudprovider.TDnsPolicyType(policy.PolicyType) == policyType && cloudprovider.IsPolicyValueEqual(
			cloudprovider.TDnsPolicyTypeValue(policy.Params),
			policyValue,
		) {
			return nil
		}
		self.RemovePolicy(ctx, userCred, policy.Id)
	}

	policy, err = DnsTrafficPolicyManager.Register(ctx, userCred, provider, policyType, policyValue)
	if err != nil {
		return errors.Wrapf(err, "DnsTrafficPolicyManager.Register")
	}

	recordPolicy := &SDnsRecordSetTrafficPolicy{}
	recordPolicy.SetModelManager(DnsRecordSetTrafficPolicyManager, recordPolicy)
	recordPolicy.DnsRecordsetId = self.Id
	recordPolicy.TrafficPolicyId = policy.Id

	return DnsRecordSetTrafficPolicyManager.TableSpec().Insert(ctx, recordPolicy)
}
