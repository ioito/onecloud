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

package cloudprovider

import (
	"fmt"
	"sort"
	"strings"

	"yunion.io/x/jsonutils"
)

type TDnsZoneType string
type TDnsPolicyType string

const (
	PublicZone  = TDnsZoneType("PublicZone")
	PrivateZone = TDnsZoneType("PrivateZone")
)

const (
	DnsPolicyTypeSimple         = TDnsPolicyType("Simple")         //简单
	DnsPolicyTypeByCarrier      = TDnsPolicyType("ByCarrier")      //运营商
	DnsPolicyTypeByGeoLocation  = TDnsPolicyType("ByGeoLocation")  //地理区域
	DnsPolicyTypeBySearchEngine = TDnsPolicyType("BySearchEngine") //搜索引擎
	DnsPolicyTypeIpRange        = TDnsPolicyType("IpRange")        //自定义IP范围
	DnsPolicyTypeWeighted       = TDnsPolicyType("Weighted")       //加权
)

type SDnsZoneCreateOptions struct {
}

type SDnsTrafficPolicySetOptions struct {
}

type SAddDnsRecordSetOptions struct {
}

type SRemoveDnsRecordSetOptions struct {
}

type DnsRecordSet struct {
	Id      string
	iRecord ICloudDnsRecordSet

	DnsName     string
	DnsType     string
	DnsValue    string
	Status      string
	Ttl         int
	PolicyType  TDnsPolicyType
	PolicyParms *jsonutils.JSONDict
}

func (record DnsRecordSet) Equals(r DnsRecordSet) bool {
	if record.DnsName != r.DnsName {
		return false
	}
	if record.DnsType != r.DnsType {
		return false
	}
	if record.DnsValue != r.DnsValue {
		return false
	}
	if record.Ttl != r.Ttl {
		return false
	}
	if record.PolicyType != r.PolicyType {
		return false
	}
	{
		if record.PolicyParms != nil && !record.PolicyParms.Equals(r.PolicyParms) {
			return false
		}
		if r.PolicyParms != nil && !r.PolicyParms.Equals(record.PolicyParms) {
			return false
		}
	}
	return true
}

func (record DnsRecordSet) String() string {
	if record.PolicyParms != nil {
		return fmt.Sprintf("%s-%s-%s-%s-%s", record.DnsName, record.DnsType, record.DnsValue, record.PolicyType, record.PolicyParms.String())
	}
	return fmt.Sprintf("%s-%s-%s-%s", record.DnsName, record.DnsType, record.DnsValue, record.PolicyType)
}

type DnsRecordSets []DnsRecordSet

func (records DnsRecordSets) Len() int {
	return len(records)
}

func (records DnsRecordSets) Swap(i, j int) {
	records[i], records[j] = records[j], records[i]
}

func (records DnsRecordSets) Less(i, j int) bool {
	if records[i].DnsName < records[j].DnsName {
		return true
	}
	if records[i].DnsType < records[j].DnsType {
		return true
	}
	if records[i].DnsValue < records[j].DnsValue {
		return true
	}
	if records[i].PolicyType < records[j].PolicyType {
		return true
	}
	if records[i].PolicyParms != nil && records[j].PolicyParms != nil && records[i].PolicyParms.String() < records[j].PolicyParms.String() {
		return true
	}
	return false
}

func CompareDnsRecordSet(iRecords []ICloudDnsRecordSet, local []DnsRecordSet) ([]DnsRecordSet, []DnsRecordSet, []DnsRecordSet, []DnsRecordSet) {
	remote := DnsRecordSets{}
	for i := range iRecords {
		remote = append(remote, DnsRecordSet{
			iRecord:     iRecords[i],
			DnsName:     iRecords[i].GetDnsName(),
			DnsType:     iRecords[i].GetDnsType(),
			DnsValue:    iRecords[i].GetDnsValue(),
			Status:      iRecords[i].GetStatus(),
			Ttl:         iRecords[i].GetTTL(),
			PolicyType:  iRecords[i].GetPolicyType(),
			PolicyParms: iRecords[i].GetPolicyParams(),
		})
	}
	sort.Sort(remote)
	sort.Sort(DnsRecordSets(local))
	common, add, del, update := []DnsRecordSet{}, []DnsRecordSet{}, []DnsRecordSet{}, []DnsRecordSet{}
	i, j := 0, 0
	for i < len(local) || j < len(remote) {
		if i < len(local) && j < len(remote) {
			l, r := local[i].String(), remote[j].String()
			cmp := strings.Compare(l, r)
			if cmp == 0 {
				remote[j].Id = local[i].Id
				if local[i].Equals(remote[j]) {
					common = append(common, remote[j])
				} else {
					update = append(update, remote[j])
				}
				i++
				j++
			} else if cmp < 0 {
				del = append(del, remote[j])
				j++
			} else {
				add = append(add, local[i])
				i++
			}
		} else if i >= len(local) {
			del = append(del, remote[j])
			j++
		} else if j >= len(remote) {
			add = append(add, local[i])
			i++
		}
	}
	return common, add, del, update
}
