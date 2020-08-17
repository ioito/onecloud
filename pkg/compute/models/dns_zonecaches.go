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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SDnsZoneCacheManager struct {
	db.SStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SDnsZoneResourceBaseManager
}

var DnsZoneCacheManager *SDnsZoneCacheManager

func init() {
	DnsZoneCacheManager = &SDnsZoneCacheManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SDnsZoneCache{},
			"dns_zone_caches_tbl",
			"dns_zonecache",
			"dns_zonecaches",
		),
	}
	DnsZoneCacheManager.SetVirtualObject(DnsZoneCacheManager)

}

type SDnsZoneCache struct {
	db.SStandaloneResourceBase
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
	q, err = manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
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
	stdRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.DnsZoneCacheDetails{
			StandaloneResourceDetails: stdRows[i],
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
	return self.Delete(ctx, userCred)
}
