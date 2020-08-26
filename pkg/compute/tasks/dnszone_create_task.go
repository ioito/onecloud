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

package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DnsZoneCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DnsZoneCreateTask{})
}

func (self *DnsZoneCreateTask) taskFailed(ctx context.Context, dnsZone *models.SDnsZone, err error) {
	dnsZone.SetStatus(self.GetUserCred(), api.DNS_ZONE_STATUS_CREATE_FAILE, err.Error())
	db.OpsLog.LogEvent(dnsZone, db.ACT_CREATE, dnsZone.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, dnsZone, logclient.ACT_CREATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DnsZoneCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	dnsZone := obj.(*models.SDnsZone)

	accountId, _ := self.GetParams().GetString("cloudaccount_id")
	iVpcs := []cloudprovider.ICloudVpc{}
	switch cloudprovider.TDnsZoneType(dnsZone.ZoneType) {
	case cloudprovider.PrivateZone:
		vpcs, err := dnsZone.GetVpcs()
		if err != nil {
			self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "GetVpcs"))
			return
		}
		if len(vpcs) == 0 {
			self.taskFailed(ctx, dnsZone, fmt.Errorf("empty vpcs for dnszone"))
			return
		}
		for _, vpc := range vpcs {
			if len(vpc.ManagerId) > 0 {
				iVpc, err := vpc.GetIVpc()
				if err != nil {
					self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "GetIVpc for %s", vpc.Name))
					return
				}
				iVpcs = append(iVpcs, iVpc)
			}
		}
		if len(iVpcs) > 0 {
			account := vpcs[0].GetCloudaccount()
			if account == nil {
				self.taskFailed(ctx, dnsZone, errors.Wrapf(cloudprovider.ErrNotFound, "GetCloudaccount for vpc %s", vpcs[0].Name))
				return
			}
			accountId = account.Id
		}
	case cloudprovider.PublicZone:
		if len(accountId) == 0 {
			self.taskFailed(ctx, dnsZone, fmt.Errorf("missing cloudaccount_id"))
			return
		}
	}

	if len(accountId) > 0 || len(iVpcs) > 0 {
		cache, err := dnsZone.RegisterCache(ctx, self.GetUserCred(), accountId)
		if err != nil {
			self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "RegisterCache"))
			return
		}

		provider, err := cache.GetProvider()
		if err != nil {
			self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "GetProvider"))
			return
		}

		opts := cloudprovider.SDnsZoneCreateOptions{
			Name:     dnsZone.Name,
			Desc:     dnsZone.Description,
			ZoneType: cloudprovider.TDnsZoneType(dnsZone.ZoneType),
			Options:  dnsZone.Options,
		}
		for _, iVpc := range iVpcs {
			opts.Vpcs = append(opts.Vpcs, cloudprovider.SPrivateZoneVpc{
				Id:       iVpc.GetGlobalId(),
				RegionId: iVpc.GetRegion().GetId(),
			})
		}

		iDnsZone, err := provider.CreateICloudDnsZone(&opts)
		if err != nil {
			self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "CreateICloudDnsZone"))
			return
		}

		err = cache.SyncWithCloudDnsZone(ctx, self.GetUserCred(), iDnsZone)
		if err != nil {
			self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "SyncWithCloudDnsZone"))
			return
		}

		logclient.AddActionLogWithContext(ctx, cache, logclient.ACT_CREATE, nil, self.UserCred, true)
	}

	self.SetStage("OnSyncRecordSetComplete", nil)
	dnsZone.StartDnsZoneSyncRecordSetsTask(ctx, self.GetUserCred(), self.GetTaskId())
}

func (self *DnsZoneCreateTask) OnSyncRecordSetComplete(ctx context.Context, dnsZone *models.SDnsZone, data jsonutils.JSONObject) {
	dnsZone.SetStatus(self.GetUserCred(), api.DNS_ZONE_STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}

func (self *DnsZoneCreateTask) OnSyncRecordSetCompleteFailed(ctx context.Context, dnsZone *models.SDnsZone, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
