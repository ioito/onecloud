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
	"time"

	"gopkg.in/fatih/set.v0"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
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

	IsDirty bool `nullable:"false" default:"false"`

	ZoneType string              `width:"32" charset:"ascii" nullable:"false" list:"domain" create:"domain_required"`
	Options  *jsonutils.JSONDict `get:"domain" list:"domain" create:"domain_optional"`
}

// 创建
func (manager *SDnsZoneManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.DnsZoneCreateInput) (api.DnsZoneCreateInput, error) {
	if !regutils.MatchDomainName(input.Name) {
		return input, httperrors.NewInputParameterError("invalid domain name %s", input.Name)
	}
	if len(input.ZoneType) == 0 {
		return input, httperrors.NewMissingParameterError("zone_type")
	}
	switch cloudprovider.TDnsZoneType(input.ZoneType) {
	case cloudprovider.PrivateZone:
		vpcIds, managerId := []string{}, ""
		for _, vpcId := range input.VpcIds {
			_vpc, err := VpcManager.FetchByIdOrName(userCred, vpcId)
			if err != nil {
				if errors.Cause(err) == sql.ErrNoRows {
					return input, httperrors.NewResourceNotFoundError2("vpc", vpcId)
				}
				return input, httperrors.NewGeneralError(err)
			}
			vpc := _vpc.(*SVpc)
			if len(managerId) == 0 {
				managerId = vpc.ManagerId
			}
			account := vpc.GetCloudaccount()
			if account != nil {
				factory, err := account.GetProviderFactory()
				if err != nil {
					return input, httperrors.NewGeneralError(errors.Wrapf(err, "GetProviderFactory"))
				}
				dnsTypes := factory.GetSupportedDnsTypes()
				if isIn, _ := utils.InArray(cloudprovider.PrivateZone, dnsTypes); !isIn {
					return input, httperrors.NewNotSupportedError("Not support %s for vpc %s", cloudprovider.PrivateZone, vpc.Name)
				}
			}
			if managerId != vpc.ManagerId {
				return input, httperrors.NewConflictError("vpc ids not with same account")
			}
			vpcIds = append(vpcIds, vpc.GetId())
		}
		input.VpcIds = vpcIds
	case cloudprovider.PublicZone:
		if len(input.CloudaccountId) > 0 {
			account, err := CloudaccountManager.FetchByIdOrName(userCred, input.CloudaccountId)
			if err != nil {
				if errors.Cause(err) == sql.ErrNoRows {
					return input, httperrors.NewResourceNotFoundError2("cloudaccount", input.CloudaccountId)
				}
				return input, httperrors.NewGeneralError(err)
			}
			input.CloudaccountId = account.GetId()
		} else {
			input.Status = api.DNS_ZONE_STATUS_AVAILABLE
		}
	default:
		return input, httperrors.NewInputParameterError("unknown zone type %s", input.ZoneType)
	}
	return input, nil
}

func (self *SDnsZone) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	input := &api.DnsZoneCreateInput{}
	data.Unmarshal(input)
	switch cloudprovider.TDnsZoneType(input.ZoneType) {
	case cloudprovider.PrivateZone:
		for _, vpcId := range input.VpcIds {
			self.AddVpc(ctx, vpcId)
		}
		if len(input.VpcIds) > 0 {
			self.StartDnsZoneCreateTask(ctx, userCred, "", "")
			return
		}
	case cloudprovider.PublicZone:
		if len(input.CloudaccountId) > 0 {
			self.StartDnsZoneCreateTask(ctx, userCred, input.CloudaccountId, "")
			return
		}
	}
}

func (self *SDnsZone) StartDnsZoneCreateTask(ctx context.Context, userCred mcclient.TokenCredential, accountId, parentTaskId string) error {
	params := jsonutils.NewDict()
	if len(accountId) > 0 {
		params.Add(jsonutils.NewString(accountId), "cloudaccount_id")
	}
	task, err := taskman.TaskManager.NewTask(ctx, "DnsZoneCreateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.DNS_ZONE_STATUS_CREATING, "")
	task.ScheduleRun(nil)
	return nil
}

func (manager *SDnsZoneManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsDomainAllowList(userCred, manager)
}

// 列表
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

	_, err = dnsZone.newCache(ctx, userCred, account.Id, ext.GetGlobalId())
	if err != nil {
		return nil, errors.Wrapf(err, "newCache")
	}

	return dnsZone, nil
}

func (self *SDnsZone) GetDnsZoneCache(accountId string) (*SDnsZoneCache, error) {
	caches := []SDnsZoneCache{}
	q := DnsZoneCacheManager.Query().Equals("cloudaccount_id", accountId).Equals("dns_zone_id", self.Id)
	err := db.FetchModelObjects(DnsZoneCacheManager, q, &caches)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	if len(caches) == 1 {
		return &caches[0], nil
	}
	if len(caches) == 0 {
		return nil, sql.ErrNoRows
	}
	return nil, sqlchemy.ErrDuplicateEntry
}

func (self *SDnsZone) GetDnsZoneCaches() ([]SDnsZoneCache, error) {
	caches := []SDnsZoneCache{}
	q := DnsZoneCacheManager.Query().Equals("dns_zone_id", self.Id)
	err := db.FetchModelObjects(DnsZoneCacheManager, q, &caches)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return caches, nil
}

func (self *SDnsZone) RegisterCache(ctx context.Context, userCred mcclient.TokenCredential, accountId string) (*SDnsZoneCache, error) {
	lockman.LockRawObject(ctx, self.Keyword(), fmt.Sprintf("%s-%s", accountId, self.Id))
	defer lockman.ReleaseRawObject(ctx, self.Keyword(), fmt.Sprintf("%s-%s", accountId, self.Id))

	cache, err := self.GetDnsZoneCache(accountId)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, err
	}
	if cache != nil {
		return cache, nil
	}

	return self.newCache(ctx, userCred, accountId, "")
}

func (self *SDnsZone) newCache(ctx context.Context, userCred mcclient.TokenCredential, accountId, externalId string) (*SDnsZoneCache, error) {
	cache := &SDnsZoneCache{}
	cache.SetModelManager(DnsZoneCacheManager, cache)
	cache.Name = self.Name
	cache.ExternalId = externalId
	cache.CloudaccountId = accountId
	cache.DnsZoneId = self.Id
	err := DnsZoneCacheManager.TableSpec().Insert(ctx, cache)
	if err != nil {
		return nil, errors.Wrapf(err, "dnsZoneCache.Insert")
	}
	return cache, nil
}

func (self *SDnsZone) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDnsZoneDeleteTask(ctx, userCred, false, "")
}

func (self *SDnsZone) StartDnsZoneDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, purge bool, parentTaskId string) error {
	params := jsonutils.Marshal(map[string]bool{"purge": purge}).(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "DnsZoneDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.DNS_ZONE_STATUS_REMOVE_VPCS, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SDnsZone) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SDnsZone) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	records, err := self.GetDnsRecordSets()
	if err != nil {
		return errors.Wrapf(err, "GetDnsRecordSets for %s(%s)", self.Name, self.Id)
	}
	for i := range records {
		err = records[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "Delete record %s(%s)", records[i].Name, records[i].Id)
		}
	}
	return self.SEnabledStatusInfrasResourceBase.Delete(ctx, userCred)
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
	q := DnsRecordSetManager.Query().Equals("dns_zone_id", self.Id)
	err := db.FetchModelObjects(DnsRecordSetManager, q, &records)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return records, nil
}

func (self *SDnsZone) SyncDnsRecordSets(ctx context.Context, userCred mcclient.TokenCredential, provider string, ext cloudprovider.ICloudDnsZone) compare.SyncResult {
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
		record := cloudprovider.DnsRecordSet{
			Id:       dbRecords[i].Id,
			DnsName:  dbRecords[i].Name,
			Enabled:  dbRecords[i].Enabled.Bool(),
			Status:   dbRecords[i].Status,
			DnsType:  cloudprovider.TDnsType(dbRecords[i].DnsType),
			DnsValue: dbRecords[i].DnsValue,
			Ttl:      dbRecords[i].TTL,
		}
		record.PolicyType, record.PolicyParams, err = dbRecords[i].GetDefaultDnsTrafficPolicy(provider)
		if err != nil {
			result.Error(errors.Wrapf(err, "GetDefaultDnsTrafficPolicy(%s)", provider))
			return result
		}
		local = append(local, record)
	}

	_, del, add, update := cloudprovider.CompareDnsRecordSet(iRecords, local, false)
	for i := range add {
		_, err := self.newFromCloudDnsRecordSet(ctx, userCred, provider, add[i])
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
			err = record.syncWithCloudDnsRecord(ctx, userCred, provider, update[i])
			if err != nil {
				result.UpdateError(errors.Wrapf(err, "syncWithCloudDnsRecord"))
				continue
			}
			result.Update()
		}
	}

	return result
}

func (self *SDnsZone) newFromCloudDnsRecordSet(ctx context.Context, userCred mcclient.TokenCredential, provider string, ext cloudprovider.DnsRecordSet) (*SDnsRecordSet, error) {
	record := &SDnsRecordSet{}
	record.SetModelManager(DnsRecordSetManager, record)
	record.DnsZoneId = self.Id
	record.Name = ext.DnsName
	record.Status = ext.Status
	record.Enabled = tristate.NewFromBool(ext.Enabled)
	record.TTL = ext.Ttl
	record.DnsType = string(ext.DnsType)
	record.DnsValue = ext.DnsValue

	err := DnsRecordSetManager.TableSpec().Insert(ctx, record)
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	record.setTrafficPolicy(ctx, userCred, provider, ext.PolicyType, ext.PolicyParams)
	return record, nil
}

func (self *SDnsZone) AllowPerformSyncRecordSets(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "sync-record-sets")
}

// 同步解析列表到云上
func (self *SDnsZone) PerformSyncRecordSets(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DnsZoneSyncRecordSetsInput) (jsonutils.JSONObject, error) {
	if self.Status != api.DNS_ZONE_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("can not sync record sets in %s", self.Status)
	}
	return nil, self.StartDnsZoneSyncRecordSetsTask(ctx, userCred, "")
}

func (self *SDnsZone) DoSyncRecords(ctx context.Context, userCred mcclient.TokenCredential) error {
	_, err := db.Update(self, func() error {
		self.IsDirty = true
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	time.AfterFunc(10*time.Second, func() {
		self.DelaySync(context.Background(), userCred)
	})
	return nil
}

func (self *SDnsZone) DelaySync(ctx context.Context, userCred mcclient.TokenCredential) {
	needSync := false

	func() {
		lockman.LockObject(ctx, self)
		defer lockman.ReleaseObject(ctx, self)

		if self.IsDirty {
			_, err := db.Update(self, func() error {
				self.IsDirty = false
				return nil
			})
			if err != nil {
				log.Errorf("Update dns zone error: %s", err.Error())
			}
			needSync = true
		}
	}()

	if needSync {
		self.StartDnsZoneSyncRecordSetsTask(ctx, userCred, "")
	}
}

func (self *SDnsZone) StartDnsZoneSyncRecordSetsTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "DnsZoneSyncRecordSetsTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.DNS_ZONE_STATUS_SYNC_RECORD_SETS, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SDnsZone) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "syncstatus")
}

// 同步状态
func (self *SDnsZone) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DnsZoneSyncStatusInput) (jsonutils.JSONObject, error) {
	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "DnsZoneSyncstatusTask", "")
}

func (self *SDnsZone) AllowPerformCache(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "cache")
}

// 指定云账号创建
func (self *SDnsZone) PerformCache(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DnsZoneCacheInput) (jsonutils.JSONObject, error) {
	if self.Status != api.DNS_ZONE_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("dns zone can not cache in status %s", self.Status)
	}
	if cloudprovider.TDnsZoneType(self.ZoneType) != cloudprovider.PublicZone {
		return nil, httperrors.NewUnsupportOperationError("Only %s support cache for account", cloudprovider.PublicZone)
	}
	if len(input.CloudaccountId) == 0 {
		return nil, httperrors.NewMissingParameterError("cloudaccount_id")
	}
	account, err := CloudaccountManager.FetchByIdOrName(userCred, input.CloudaccountId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2("cloudaccount", input.CloudaccountId)
		}
		return nil, httperrors.NewGeneralError(err)
	}
	cache, err := self.RegisterCache(ctx, userCred, account.GetId())
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "RegisterCache"))
	}
	if len(cache.ExternalId) > 0 {
		return nil, httperrors.NewConflictError("account %s has been cached", account.GetName())
	}
	return nil, cache.StartDnsZoneCacheCreateTask(ctx, userCred, "")
}

func (self *SDnsZone) AllowPerformUncache(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "uncache")
}

// 删除云账号的云上资源
func (self *SDnsZone) PerformUnache(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DnsZoneUnacheInput) (jsonutils.JSONObject, error) {
	if self.Status != api.DNS_ZONE_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("dns zone can not uncache in status %s", self.Status)
	}
	if cloudprovider.TDnsZoneType(self.ZoneType) != cloudprovider.PublicZone {
		return nil, httperrors.NewUnsupportOperationError("Only %s support cache for account", cloudprovider.PublicZone)
	}
	account, err := CloudaccountManager.FetchByIdOrName(userCred, input.CloudaccountId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2("cloudaccount", input.CloudaccountId)
		}
		return nil, httperrors.NewGeneralError(err)
	}
	cache, err := self.RegisterCache(ctx, userCred, account.GetId())
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "RegisterCache"))
	}
	return nil, cache.StartDnsZoneCacheDeleteTask(ctx, userCred, "")
}

func (self *SDnsZone) AllowPerformAddVpcs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "add-vpcs")
}

// 添加VPC
func (self *SDnsZone) PerformAddVpcs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DnsZoneAddVpcsInput) (jsonutils.JSONObject, error) {
	if self.Status != api.DNS_ZONE_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("dns zone can not uncache in status %s", self.Status)
	}
	if cloudprovider.TDnsZoneType(self.ZoneType) != cloudprovider.PrivateZone {
		return nil, httperrors.NewUnsupportOperationError("Only %s support cache for account", cloudprovider.PrivateZone)
	}
	managerId := ""
	vpcs, err := self.GetVpcs()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetVpcs"))
	}
	if len(vpcs) > 0 {
		managerId = vpcs[0].ManagerId
	}
	localVpcIds := []string{}
	for _, vpc := range vpcs {
		localVpcIds = append(localVpcIds, vpc.Id)
	}

	if len(input.VpcIds) == 0 {
		return nil, httperrors.NewMissingParameterError("vpc_ids")
	}

	for _, vpcId := range input.VpcIds {
		_vpc, err := VpcManager.FetchById(vpcId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("vpc", vpcId)
			}
			return nil, httperrors.NewGeneralError(errors.Wrapf(err, "VpcManager.FetchById(%s)", vpcId))
		}
		vpc := _vpc.(*SVpc)
		if managerId != vpc.ManagerId && len(vpcs) != 0 {
			return nil, httperrors.NewConflictError("vpc %s not with same account", vpcId)
		}
		managerId = vpc.ManagerId
		if utils.IsInStringArray(vpc.Id, localVpcIds) {
			return nil, httperrors.NewConflictError("vpc %s has already in this dns zone", vpc.Id)
		}
	}
	return nil, self.StartDnsZoneAddVpcsTask(ctx, userCred, input.VpcIds, "")
}

func (self *SDnsZone) StartDnsZoneAddVpcsTask(ctx context.Context, userCred mcclient.TokenCredential, vpcIds []string, parentTaskId string) error {
	params := jsonutils.Marshal(map[string][]string{"vpc_ids": vpcIds}).(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "DnsZoneAddVpcsTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.DNS_ZONE_STATUS_ADD_VPCS, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SDnsZone) AllowPerformRemoveVpcs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "remove-vpcs")
}

// 移除VPC
func (self *SDnsZone) PerformRemoveVpcs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DnsZoneRemoveVpcsInput) (jsonutils.JSONObject, error) {
	if self.Status != api.DNS_ZONE_STATUS_AVAILABLE {
		return nil, httperrors.NewInvalidStatusError("dns zone can not uncache in status %s", self.Status)
	}
	if cloudprovider.TDnsZoneType(self.ZoneType) != cloudprovider.PrivateZone {
		return nil, httperrors.NewUnsupportOperationError("Only %s support cache for account", cloudprovider.PrivateZone)
	}
	vpcs, err := self.GetVpcs()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetVpcs"))
	}
	vpcIds := []string{}
	for _, vpc := range vpcs {
		vpcIds = append(vpcIds, vpc.Id)
	}
	for _, vpcId := range input.VpcIds {
		if utils.IsInStringArray(vpcId, vpcIds) {
			return nil, httperrors.NewResourceNotFoundError("vpc %s not in dns zone", vpcId)
		}
	}
	return nil, self.StartDnsZoneRemoveVpcsTask(ctx, userCred, input.VpcIds, "")
}

func (self *SDnsZone) StartDnsZoneRemoveVpcsTask(ctx context.Context, userCred mcclient.TokenCredential, vpcIds []string, parentTaskId string) error {
	params := jsonutils.Marshal(map[string][]string{"vpc_ids": vpcIds}).(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "DnsZoneRemoveVpcsTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.DNS_ZONE_STATUS_REMOVE_VPCS, "")
	task.ScheduleRun(nil)
	return nil
}

func (manager *SDnsZoneManager) GetPropertyCapability(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(cloudprovider.GetDnsCapabilities()), nil
}

func (self *SDnsZone) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "purge")
}

func (self *SDnsZone) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DnsZonePurgeInput) (jsonutils.JSONObject, error) {
	return nil, self.StartDnsZoneDeleteTask(ctx, userCred, true, "")
}
