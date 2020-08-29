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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DnsZoneRemoveVpcsTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DnsZoneRemoveVpcsTask{})
}

func (self *DnsZoneRemoveVpcsTask) taskFailed(ctx context.Context, dnsZone *models.SDnsZone, err error) {
	dnsZone.SetStatus(self.GetUserCred(), api.DNS_ZONE_STATUS_REMOVE_VPCS_FAILED, err.Error())
	db.OpsLog.LogEvent(dnsZone, db.ACT_REMOVE_VPCS, dnsZone.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, dnsZone, logclient.ACT_REMOVE_VPCS, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DnsZoneRemoveVpcsTask) taskComplete(ctx context.Context, dnsZone *models.SDnsZone) {
	vpcIds := self.getVpcIds()
	db.OpsLog.LogEvent(dnsZone, db.ACT_REMOVE_VPCS, dnsZone.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, dnsZone, logclient.ACT_REMOVE_VPCS, map[string][]string{"vpc_ids": vpcIds}, self.UserCred, true)
	dnsZone.SetStatus(self.GetUserCred(), api.DNS_ZONE_STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}

func (self *DnsZoneRemoveVpcsTask) getVpcIds() []string {
	vpcIds := []string{}
	self.GetParams().Unmarshal(&vpcIds, "vpc_ids")
	return vpcIds
}

func (self *DnsZoneRemoveVpcsTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	dnsZone := obj.(*models.SDnsZone)

	caches, err := dnsZone.GetDnsZoneCaches()
	if err != nil {
		self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "GetDnsZoneCaches"))
		return
	}

	vpcIds := self.getVpcIds()
	for i := range caches {
		if len(caches[i].ExternalId) == 0 {
			for _, vpcId := range vpcIds {
				err = dnsZone.RemoveVpc(ctx, vpcId)
				if err != nil {
					self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "dnsZone.RemoveVpc(%s)", vpcId))
					return
				}
			}
			self.taskComplete(ctx, dnsZone)
			return
		}
		iDnsZone, err := caches[i].GetICloudDnsZone()
		if err != nil {
			self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "GetICloudDnsZone"))
			return
		}
		for _, vpcId := range vpcIds {
			_vpc, err := models.VpcManager.FetchById(vpcId)
			if err != nil {
				self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "VpcManager.FetchById(%s)", vpcId))
				return
			}
			vpc := _vpc.(*models.SVpc)
			iVpc, err := vpc.GetIVpc()
			if err != nil {
				self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "GetIVpc for vpc %s", vpc.Name))
				return
			}
			opts := cloudprovider.SPrivateZoneVpc{
				Id:       iVpc.GetGlobalId(),
				RegionId: iVpc.GetRegion().GetId(),
			}
			err = iDnsZone.RemoveVpc(&opts)
			if err != nil {
				self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "iDnsZone.RemoveVpc(%s,%s)", opts.Id, opts.RegionId))
				return
			}
			err = dnsZone.RemoveVpc(ctx, vpcId)
			if err != nil {
				self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "dnsZone.RemoveVpc(%s)", vpcId))
				return
			}
		}
	}

	for _, vpcId := range vpcIds { // for kvm
		err = dnsZone.RemoveVpc(ctx, vpcId)
		if err != nil {
			self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "dnsZone.RemoveVpc(%s)", vpcId))
			return
		}
	}

	self.taskComplete(ctx, dnsZone)
}
