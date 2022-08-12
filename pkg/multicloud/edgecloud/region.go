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
	"net/url"
)

type SRegion struct {
	cli *SEdgeCloudClient
}

func (self *SEdgeCloudClient) GetRegions(pageSize, pageNumber int) ([]SRegion, int, error) {
	params := url.Values{}
	//params.Set("pageSize", fmt.Sprintf("%d", pageSize))
	//params.Set("platformType", "cloudst_prov_edge")
	//params.Set("platformCode", "VMWARE")
	//params.Set("pageNum", fmt.Sprintf("%d", pageNumber))
	regions := []SRegion{}
	total, err := self.list("region/regions", params, &regions)
	if err != nil {
		return nil, 0, err
	}
	return regions, total, nil
}

func (self *SRegion) GetClient() *SEdgeCloudClient {
	return self.cli
}
