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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
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

type SDnsZoneCacheManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SDnsZoneResourceBaseManager
}

var DnsZoneCacheManager *SDnsZoneCacheManager

func init() {
	DnsZoneCacheManager = &SDnsZoneCacheManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SDnsZoneCache{},
			"dns_zone_caches_tbl",
			"dns_zonecache",
			"dns_zonecaches",
		),
	}
	DnsZoneCacheManager.SetVirtualObject(DnsZoneCacheManager)

}

type SDnsZoneCache struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SDnsZoneResourceBase

	// 归属云账号ID
	CloudaccountId string `width:"36" charset:"ascii" nullable:"false" list:"user"`
}

func (manager *SDnsZoneCacheManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.DnsZoneCacheCreateInput) (api.DnsZoneCacheCreateInput, error) {
	return input, httperrors.NewNotSupportedError("Not support")
}

func (manager *SDnsZoneCacheManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsDomainAllowList(userCred, manager)
}

func (manager *SDnsZoneCacheManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DnsZoneCacheListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (self *SDnsZoneCache) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.DnsZoneCacheDetails, error) {
	return api.DnsZoneCacheDetails{}, nil
}

func (manager *SDnsZoneCacheManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.DnsZoneCacheDetails {
	rows := make([]api.DnsZoneCacheDetails, len(objs))
	stdRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.DnsZoneCacheDetails{
			StatusStandaloneResourceDetails: stdRows[i],
		}
	}
	return rows
}

func (self *SDnsZoneCache) GetDnsZone() (*SDnsZone, error) {
	dnsZone, err := DnsZoneManager.FetchById(self.DnsZoneId)
	if err != nil {
		return nil, errors.Wrapf(err, "DnsZoneManager.FetchById(%s)", self.DnsZoneId)
	}
	return dnsZone.(*SDnsZone), nil
}

func (self *SDnsZoneCache) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	dnsZone, err := self.GetDnsZone()
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return errors.Wrapf(err, "GetDnsZone for %s", self.Name)
		}
		return self.Delete(ctx, userCred)
	}
	if cloudprovider.TDnsZoneType(dnsZone.ZoneType) == cloudprovider.PublicZone {
		return self.Delete(ctx, userCred)
	}

	err = dnsZone.RealDelete(ctx, userCred)
	if err != nil {
		return errors.Wrapf(err, "dnsZone.RealDelete for %s(%s)", dnsZone.Name, dnsZone.Id)
	}

	return self.Delete(ctx, userCred)
}

func (self *SDnsZoneCache) SyncWithCloudDnsZone(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudDnsZone) error {
	_, err := db.Update(self, func() error {
		self.ExternalId = ext.GetGlobalId()
		self.Status = ext.GetStatus()
		self.Name = ext.GetName()
		return nil
	})
	return err
}

func (self *SDnsZoneCache) GetCloudaccount() (*SCloudaccount, error) {
	account, err := CloudaccountManager.FetchById(self.CloudaccountId)
	if err != nil {
		return nil, errors.Wrapf(err, "loudaccountManager.FetchById(%s)", self.CloudaccountId)
	}
	return account.(*SCloudaccount), nil
}

func (self *SDnsZoneCache) GetProvider() (cloudprovider.ICloudProvider, error) {
	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCloudaccount")
	}
	return account.GetProvider()
}

func (self *SDnsZoneCache) GetICloudDnsZone() (cloudprovider.ICloudDnsZone, error) {
	provider, err := self.GetProvider()
	if err != nil {
		return nil, errors.Wrapf(err, "GetProvider")
	}
	return provider.GetICloudDnsZoneById(self.ExternalId)
}

func (self *SDnsZoneCache) StartDnsZoneCacheDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return errors.Wrapf(err, "GetDnsZone")
	}
	dnsZone.SetStatus(userCred, api.DNS_ZONE_STATUS_UNCACHING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DnsZoneCacheDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.DNS_ZONE_CACHE_STATUS_DELETING, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SDnsZoneCache) StartDnsZoneCacheCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return errors.Wrapf(err, "GetDnsZone")
	}
	dnsZone.SetStatus(userCred, api.DNS_ZONE_STATUS_CACHING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "DnsZoneCacheCreateTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.DNS_ZONE_CACHE_STATUS_CREATING, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SDnsZoneCache) GetRecordSetsByDnsType(dnsTypes []cloudprovider.TDnsType) ([]SDnsRecordSet, error) {
	records := []SDnsRecordSet{}
	q := DnsRecordSetManager.Query().Equals("dns_zone_id", self.DnsZoneId).In("dns_type", dnsTypes)
	err := db.FetchModelObjects(DnsRecordSetManager, q, &records)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return records, nil
}

func (self *SDnsZoneCache) GetRecordSets() ([]cloudprovider.DnsRecordSet, error) {
	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCloudaccount")
	}
	factory, err := account.GetProviderFactory()
	if err != nil {
		return nil, errors.Wrapf(err, "GetProviderFactory")
	}
	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return nil, errors.Wrapf(err, "GetDnsZone")
	}

	_dnsTypes := factory.GetSupportedDnsTypes()
	dnsTypes, ok := _dnsTypes[cloudprovider.TDnsZoneType(dnsZone.ZoneType)]
	if !ok {
		return nil, fmt.Errorf("invalid zone type %s for %s(%s)", dnsZone.ZoneType, account.Name, account.Provider)
	}
	records, err := self.GetRecordSetsByDnsType(dnsTypes)
	if err != nil {
		return nil, errors.Wrapf(err, "GetRecordSetsByDnsType")
	}
	ret := []cloudprovider.DnsRecordSet{}
	for i := range records {
		record := cloudprovider.DnsRecordSet{
			Id:       records[i].Id,
			DnsName:  records[i].Name,
			DnsValue: records[i].DnsValue,
			DnsType:  cloudprovider.TDnsType(records[i].DnsType),
			Enabled:  records[i].Enabled.Bool(),
			Status:   records[i].Status,
			Ttl:      records[i].TTL,
		}

		record.PolicyType, record.PolicyParams, err = records[i].GetDefaultDnsTrafficPolicy(account.Provider)
		if err != nil {
			return nil, errors.Wrapf(err, "GetDefaultDnsTrafficPolicy(%s)", account.Provider)
		}
		ret = append(ret, record)
	}

	return ret, nil
}

func (self *SDnsZoneCache) SyncRecordSets(ctx context.Context, userCred mcclient.TokenCredential) error {
	iDnsZone, err := self.GetICloudDnsZone()
	if err != nil {
		return errors.Wrapf(err, "GetICloudDnsZone")
	}
	iRecordSets, err := iDnsZone.GetIDnsRecordSets()
	if err != nil {
		return errors.Wrapf(err, "GetIDnsRecordSets")
	}
	dbRecordSets, err := self.GetRecordSets()
	if err != nil {
		return errors.Wrapf(err, "GetRecordSets")
	}

	common, add, del, update := cloudprovider.CompareDnsRecordSet(iRecordSets, dbRecordSets)
	log.Infof("sync %s records for cloud common: %d add: %d del: %d update: %d", self.Name, len(common), len(add), len(del), len(update))
	return iDnsZone.SyncDnsRecordSets(common, add, del, update)
}

func (manager *SDnsZoneCacheManager) removeCaches(ctx context.Context, userCred mcclient.TokenCredential, accountId string) error {
	q := manager.Query().Equals("cloudaccount_id", accountId)
	caches := []SDnsZoneCache{}
	err := db.FetchModelObjects(manager, q, &caches)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range caches {
		err = caches[i].syncRemove(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "syncRemove cache %s(%s)", caches[i].Name, caches[i].Id)
		}
	}
	return nil
}
