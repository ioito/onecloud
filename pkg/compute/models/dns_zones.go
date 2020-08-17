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

	"gopkg.in/fatih/set.v0"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SDnsZoneManager struct {
	db.SEnabledStatusInfrasResourceBaseManager
}

var DnsZoneManager *SDnsZoneManager

func init() {
	DnsZoneManager = &SDnsZoneManager{
		SEnabledStatusInfrasResourceBaseManager: db.NewEnabledStatusInfrasResourceBaseManager(
			SDnsZone{},
			"dns_zones_tbl",
			"dns_zone",
			"dns_zones",
		),
	}
	DnsZoneManager.SetVirtualObject(DnsZoneManager)

}

type SDnsZone struct {
	db.SEnabledStatusInfrasResourceBase

	ZoneType string              `width:"32" charset:"ascii" nullable:"false" list:"domain" create:"domain_required"`
	Options  *jsonutils.JSONDict `get:"domain" list:"domain" create:"domain_optional"`
}

// 创建
func (manager *SDnsZoneManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.DnsZoneCreateInput) (api.DnsZoneCreateInput, error) {
	if len(input.ZoneType) == 0 {
		return input, httperrors.NewMissingParameterError("zone_type")
	}
	if !utils.IsInStringArray(input.ZoneType, []string{string(cloudprovider.PublicZone), string(cloudprovider.PrivateZone)}) {
		return input, httperrors.NewInputParameterError("invalid zone type %s", input.ZoneType)
	}
	return input, nil
}

func (manager *SDnsZoneManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsDomainAllowList(userCred, manager)
}

// 解析列表
func (manager *SDnsZoneManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DnsZoneListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SEnabledStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, err
	}
	if len(query.ZoneType) > 0 {
		q = q.Equals("zone_type", query.ZoneType)
	}
	return q, nil
}

func (self *SDnsZone) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsDomainAllowUpdate(userCred, self)
}

// 解析详情
func (self *SDnsZone) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.DnsZoneDetails, error) {
	return api.DnsZoneDetails{}, nil
}

func (manager *SDnsZoneManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.DnsZoneDetails {
	rows := make([]api.DnsZoneDetails, len(objs))
	enRows := manager.SEnabledStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.DnsZoneDetails{
			EnabledStatusInfrasResourceBaseDetails: enRows[i],
		}
	}
	return rows
}

func (self *SDnsZone) RemoveVpc(ctx context.Context, vpcId string) error {
	q := DnsZoneVpcManager.Query().Equals("dns_zone_id", self.Id).Equals("vpc_id", vpcId)
	zvs := []SDnsZoneVpc{}
	err := db.FetchModelObjects(DnsZoneVpcManager, q, &zvs)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range zvs {
		err = zvs[i].Delete(ctx, nil)
		if err != nil {
			return errors.Wrap(err, "Delete")
		}
	}
	return nil
}

func (self *SDnsZone) AddVpc(ctx context.Context, vpcId string) error {
	zv := &SDnsZoneVpc{}
	zv.SetModelManager(DnsZoneVpcManager, zv)
	zv.VpcId = vpcId
	zv.DnsZoneId = self.Id
	return DnsZoneVpcManager.TableSpec().Insert(ctx, zv)
}

func (self *SDnsZone) GetVpcs() ([]SVpc, error) {
	sq := DnsZoneVpcManager.Query("vpc_id").Equals("dns_zone_id", self.Id)
	q := VpcManager.Query().In("id", sq.SubQuery())
	vpcs := []SVpc{}
	err := db.FetchModelObjects(VpcManager, q, &vpcs)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return vpcs, nil
}

func (manager *SDnsZoneManager) newFromCloudDnsZone(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudDnsZone, account *SCloudaccount) (*SDnsZone, error) {
	zoneName, zoneType, vpcIds := ext.GetName(), ext.GetZoneType(), []string{}
	switch zoneType {
	case cloudprovider.PublicZone:
		q := manager.Query().Equals("name", zoneName).Equals("zone_type", string(zoneType))
		dnsZones := []SDnsZone{}
		err := db.FetchModelObjects(manager, q, &dnsZones)
		if err != nil {
			return nil, errors.Wrapf(err, "db.FetchModelObjects")
		}
		if len(dnsZones) > 0 {
			return &dnsZones[0], nil
		}
	case cloudprovider.PrivateZone:
		externalVpcIds, err := ext.GetICloudVpcIds()
		if err != nil {
			return nil, errors.Wrapf(err, "GetICloudVpcIds")
		}
		for _, externalId := range externalVpcIds {
			vpc, err := db.FetchByExternalIdAndManagerId(VpcManager, externalId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
				sq := CloudproviderManager.Query("id").Equals("cloudaccount_id", account.Id)
				return q.In("manager_id", sq.SubQuery())
			})
			if err != nil {
				return nil, errors.Wrapf(err, "vpc.FetchByExternalIdAndManagerId(%s)", externalId)
			}
			vpcIds = append(vpcIds, vpc.GetId())
		}
	default:
		return nil, fmt.Errorf("invalid zone type %s", zoneType)
	}
	dnsZone := &SDnsZone{}
	dnsZone.SetModelManager(manager, dnsZone)
	dnsZone.Name = zoneName
	dnsZone.ZoneType = string(zoneType)
	dnsZone.Enabled = tristate.True
	dnsZone.Status = ext.GetStatus()
	dnsZone.Options = ext.GetOptions()
	err := manager.TableSpec().Insert(ctx, dnsZone)
	if err != nil {
		return nil, errors.Wrapf(err, "dnsZone.Insert")
	}

	for _, vpcId := range vpcIds {
		dnsZone.AddVpc(ctx, vpcId)
	}

	SyncCloudDomain(userCred, dnsZone, account.GetOwnerId())
	dnsZone.SyncShareState(ctx, userCred, account.getAccountShareInfo())

	cache := &SDnsZoneCache{}
	cache.SetModelManager(DnsZoneCacheManager, cache)
	cache.Name = zoneName
	cache.ExternalId = ext.GetGlobalId()
	cache.CloudaccountId = account.Id
	cache.DnsZoneId = dnsZone.Id
	err = DnsZoneCacheManager.TableSpec().Insert(ctx, cache)
	if err != nil {
		return nil, errors.Wrapf(err, "dnsZoneCache.Insert")
	}

	return dnsZone, nil
}

func (self *SDnsZone) syncWithCloudDnsZone(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudDnsZone, accountId string) error {
	_, err := db.Update(self, func() error {
		self.Options = ext.GetOptions()
		return nil
	})
	localVpcs, err := self.GetVpcs()
	if err != nil {
		return errors.Wrap(err, "GetVpcs")
	}
	vpcMaps := map[string]string{}
	local := set.New(set.ThreadSafe)
	for _, vpc := range localVpcs {
		local.Add(vpc.ExternalId)
		vpcMaps[vpc.ExternalId] = vpc.Id
	}
	externalVpcIds, err := ext.GetICloudVpcIds()
	if err != nil {
		return errors.Wrapf(err, "GetICloudVpcIds")
	}
	remote := set.New(set.ThreadSafe)
	for _, id := range externalVpcIds {
		remote.Add(id)
	}
	for _, del := range set.Difference(local, remote).List() {
		id := del.(string)
		if vpcId, ok := vpcMaps[id]; ok {
			self.RemoveVpc(ctx, vpcId)
		}
	}

	for _, add := range set.Difference(remote, local).List() {
		externalId := add.(string)
		vpc, err := db.FetchByExternalIdAndManagerId(VpcManager, externalId, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			sq := CloudproviderManager.Query("id").Equals("cloudaccount_id", accountId)
			return q.In("manager_id", sq.SubQuery())
		})
		if err != nil {
			return errors.Wrapf(err, "vpc.FetchByExternalIdAndManagerId(%s)", externalId)
		}
		self.AddVpc(ctx, vpc.GetId())
	}

	return err
}

func (self *SDnsZone) GetDnsRecordSets() ([]SDnsRecordSet, error) {
	records := []SDnsRecordSet{}
	q := DnsRecordSetManager.Query().Equals("dnszone_id", self.Id)
	err := db.FetchModelObjects(DnsRecordSetManager, q, &records)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return records, nil
}

func (self *SDnsZone) SyncDnsRecordSets(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudDnsZone) compare.SyncResult {
	lockman.LockRawObject(ctx, self.Keyword(), fmt.Sprintf("%s-records", self.Id))
	defer lockman.ReleaseRawObject(ctx, self.Keyword(), fmt.Sprintf("%s-records", self.Id))

	result := compare.SyncResult{}

	iRecords, err := ext.GetIDnsRecordSets()
	if err != nil {
		result.Error(errors.Wrapf(err, "GetIDnsRecordSets"))
		return result
	}

	dbRecords, err := self.GetDnsRecordSets()
	if err != nil {
		result.Error(errors.Wrapf(err, "GetDnsRecordSets"))
		return result
	}
	local := []cloudprovider.DnsRecordSet{}
	for i := range dbRecords {
		policy, err := dbRecords[i].GetDnsTrafficPolicy()
		if err != nil {
			result.Error(errors.Wrapf(err, "GetDnsTrafficPolicy for %s(%s)", dbRecords[i].Name, dbRecords[i].Id))
			return result
		}
		local = append(local, cloudprovider.DnsRecordSet{
			Id:          dbRecords[i].Id,
			DnsName:     dbRecords[i].Name,
			DnsType:     dbRecords[i].DnsType,
			DnsValue:    dbRecords[i].DnsValue,
			Ttl:         dbRecords[i].TTL,
			PolicyType:  cloudprovider.TDnsPolicyType(policy.PolicyType),
			PolicyParms: policy.Params,
		})
	}

	_, del, add, update := cloudprovider.CompareDnsRecordSet(iRecords, local)
	for i := range add {
		_, err := self.newFromCloudDnsRecordSet(ctx, userCred, add[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	for i := range del {
		_record, err := DnsRecordSetManager.FetchById(del[i].Id)
		if err != nil {
			result.DeleteError(errors.Wrapf(err, "DnsRecordSetManager.FetchById(%s)", del[i].Id))
			continue
		}
		record := _record.(*SDnsRecordSet)
		err = record.syncRemove(ctx, userCred)
		if err != nil {
			result.DeleteError(errors.Wrapf(err, "syncRemove"))
			continue
		}
		result.Delete()
	}

	if self.ZoneType == string(cloudprovider.PrivateZone) {
		for i := range update {
			_record, err := DnsRecordSetManager.FetchById(update[i].Id)
			if err != nil {
				result.UpdateError(errors.Wrapf(err, "DnsRecordSetManager.FetchById(%s)", del[i].Id))
				continue
			}
			record := _record.(*SDnsRecordSet)
			err = record.syncWithCloudDnsRecord(ctx, userCred, update[i])
			if err != nil {
				result.UpdateError(errors.Wrapf(err, "syncWithCloudDnsRecord"))
				continue
			}
			result.Update()
		}
	}

	return result
}

func (self *SDnsZone) newFromCloudDnsRecordSet(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.DnsRecordSet) (*SDnsRecordSet, error) {
	policy, err := DnsTrafficPolicyManager.getOrCreateTrafficPolicy(ctx, userCred, string(ext.PolicyType), ext.PolicyParms)
	if err != nil {
		return nil, errors.Wrapf(err, "etOrCreateTrafficPolicy")
	}

	record := &SDnsRecordSet{}
	record.SetModelManager(DnsRecordSetManager, record)
	record.DnsZoneId = self.Id
	record.Name = ext.DnsName
	record.Status = ext.Status
	record.TTL = ext.Ttl
	record.DnsType = ext.DnsType
	record.DnsValue = ext.DnsValue
	record.DnsTrafficePolicyId = policy.Id

	err = DnsRecordSetManager.TableSpec().Insert(ctx, record)
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	return record, nil
}
