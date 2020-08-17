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

package aws

import (
	"fmt"

	"yunion.io/x/pkg/errors"
)

type SHostedZones struct {
	HostedZone  []SHostedZone `xml:"HostedZones>HostedZone"`
	Marker      string        `xml:"Marker"`
	NextMarker  string        `xml:"NextMarker"`
	MaxItems    string        `xml:"MaxItems"`
	IsTruncated bool          `xml:"IsTruncated"`
}

type SHostedZone struct {
	client          *SAwsClient
	CallerReference string `xml:"CallerReference"`
	Id              string `xml:"Id"`
	Name            string `xml:"Name"`
}

func (self *SAwsClient) ListHostedZones(delegationSetId string, offset string, limit int) (*SHostedZones, error) {
	if limit < 1 || limit > 50 {
		limit = 50
	}
	params := map[string]string{
		"maxitems": fmt.Sprintf("%d", limit),
	}
	if len(delegationSetId) > 0 {
		params["delegationsetid"] = delegationSetId
	}
	if len(offset) > 0 {
		params["marker"] = offset
	}
	zones := &SHostedZones{}
	err := self.route53Request("ListHostedZones", params, zones)
	if err != nil {
		return nil, errors.Wrapf(err, "ListHostedZones")
	}
	return zones, nil
}
