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

package edgecloud

import (
	"fmt"
	"net/url"
)

type SLoadbalancer struct {
	InstanceId string
	SlbId      string
	SlbName    string
	RegionCode string
	PublicIp   string
	VpcId      string
	SubnetId   string
	Status     string
	Bandwidth  string
	Vip        string
}

func (self *SRegion) GetLoadbalancers(pageSize, pageNum int) ([]SLoadbalancer, int, error) {
	params := url.Values{}
	//params.Set("platformType", "VMWARE")
	//params.Set("regionCode", "region.edge_h3c.wz")
	//params.Set("groupId", self.cli.groupId)
	//params.Set("billId", self.cli.billId)
	if pageSize > 0 {
		params.Set("pageSize", fmt.Sprintf("%d", pageSize))
	}
	if pageNum > 0 {
		params.Set("pageNum", fmt.Sprintf("%d", pageNum))
	}
	lbs := []SLoadbalancer{}
	total, err := self.cli.list("slb/list", params, &lbs)
	if err != nil {
		return nil, 0, err
	}
	return lbs, total, nil
}
