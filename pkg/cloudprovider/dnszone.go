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
	"yunion.io/x/log"
)

type TDnsZoneType string
type TDnsPolicyType string

type TDnsType string

type TDnsPolicyTypeValue jsonutils.JSONObject

const (
	PublicZone  = TDnsZoneType("PublicZone")
	PrivateZone = TDnsZoneType("PrivateZone")
)

const (
	DnsPolicyTypeSimple           = TDnsPolicyType("Simple")           //简单
	DnsPolicyTypeByCarrier        = TDnsPolicyType("ByCarrier")        //运营商
	DnsPolicyTypeByGeoLocation    = TDnsPolicyType("ByGeoLocation")    //地理区域
	DnsPolicyTypeBySearchEngine   = TDnsPolicyType("BySearchEngine")   //搜索引擎
	DnsPolicyTypeIpRange          = TDnsPolicyType("IpRange")          //自定义IP范围
	DnsPolicyTypeWeighted         = TDnsPolicyType("Weighted")         //加权
	DnsPolicyTypeFailover         = TDnsPolicyType("Failover")         //故障转移
	DnsPolicyTypeMultiValueAnswer = TDnsPolicyType("MultiValueAnswer") //多值应答
	DnsPolicyTypeLatency          = TDnsPolicyType("Latency")          //延迟
)

const (
	DnsTypeA            = TDnsType("A")
	DnsTypeAAAA         = TDnsType("AAAA")
	DnsTypeCAA          = TDnsType("CAA")
	DnsTypeCNAME        = TDnsType("CNAME")
	DnsTypeMX           = TDnsType("MX")
	DnsTypeNS           = TDnsType("NS")
	DnsTypeSRV          = TDnsType("SRV")
	DnsTypeSOA          = TDnsType("SOA")
	DnsTypeTXT          = TDnsType("TXT")
	DnsTypePTR          = TDnsType("PTR")
	DnsTypeDS           = TDnsType("DS")
	DnsTypeDNSKEY       = TDnsType("DNSKEY")
	DnsTypeIPSECKEY     = TDnsType("IPSECKEY")
	DnsTypeNAPTR        = TDnsType("NAPTR")
	DnsTypeSPF          = TDnsType("SPF")
	DnsTypeSSHFP        = TDnsType("SSHFP")
	DnsTypeTLSA         = TDnsType("TLSA")
	DnsTypeREDIRECT_URL = TDnsType("REDIRECT_URL") //显性URL转发
	DnsTypeFORWARD_URL  = TDnsType("FORWARD_URL")  //隐性URL转发
)

var (
	SUPPORTED_DNS_TYPES = []TDnsType{
		DnsTypeA,
		DnsTypeAAAA,
		DnsTypeCAA,
		DnsTypeCNAME,
		DnsTypeMX,
		DnsTypeNS,
		DnsTypeSRV,
		DnsTypeSOA,
		DnsTypeTXT,
		DnsTypePTR,
		DnsTypeDS,
		DnsTypeDNSKEY,
		DnsTypeIPSECKEY,
		DnsTypeNAPTR,
		DnsTypeSPF,
		DnsTypeSSHFP,
		DnsTypeTLSA,
		DnsTypeREDIRECT_URL,
		DnsTypeFORWARD_URL,
	}
)

var (
	DnsPolicyValueNil TDnsPolicyTypeValue = nil

	DnsPolicyTypeByCarrierUnicom      = TDnsPolicyTypeValue(jsonutils.Marshal(map[string]string{"carrier": "unicom"}))
	DnsPolicyTypeByCarrierTelecom     = TDnsPolicyTypeValue(jsonutils.Marshal(map[string]string{"carrier": "telecom"}))
	DnsPolicyTypeByCarrierChinaMobile = TDnsPolicyTypeValue(jsonutils.Marshal(map[string]string{"carrier": "chinamobile"}))
	DnsPolicyTypeByCarrierCernet      = TDnsPolicyTypeValue(jsonutils.Marshal(map[string]string{"carrier": "cernet"}))

	DnsPolicyTypeByGeoLocationOversea  = TDnsPolicyTypeValue(jsonutils.Marshal(map[string]string{"location": "oversea"}))
	DnsPolicyTypeByGeoLocationMainland = TDnsPolicyTypeValue(jsonutils.Marshal(map[string]string{"location": "mainland"}))

	DnsPolicyTypeBySearchEngineBaidu   = TDnsPolicyTypeValue(jsonutils.Marshal(map[string]string{"searchengine": "baidu"}))
	DnsPolicyTypeBySearchEngineGoogle  = TDnsPolicyTypeValue(jsonutils.Marshal(map[string]string{"searchengine": "google"}))
	DnsPolicyTypeBySearchEngineBing    = TDnsPolicyTypeValue(jsonutils.Marshal(map[string]string{"searchengine": "bing"}))
	DnsPolicyTypeBySearchEngineYoudao  = TDnsPolicyTypeValue(jsonutils.Marshal(map[string]string{"searchengine": "youdao"}))
	DnsPolicyTypeBySearchEngineSousou  = TDnsPolicyTypeValue(jsonutils.Marshal(map[string]string{"searchengine": "sousou"}))
	DnsPolicyTypeBySearchEngineSougou  = TDnsPolicyTypeValue(jsonutils.Marshal(map[string]string{"searchengine": "sougou"}))
	DnsPolicyTypeBySearchEngineQihu360 = TDnsPolicyTypeValue(jsonutils.Marshal(map[string]string{"searchengine": "qihu360"}))
)

type SPrivateZoneVpc struct {
	Id       string
	RegionId string
}

type SDnsZoneCreateOptions struct {
	Name     string
	Desc     string
	ZoneType TDnsZoneType
	Vpcs     []SPrivateZoneVpc
	Options  *jsonutils.JSONDict
}

func IsSupportPolicyValue(v1 TDnsPolicyTypeValue, arr []TDnsPolicyTypeValue) bool {
	for i := range arr {
		if IsPolicyValueEqual(v1, arr[i]) {
			return true
		}
	}
	return false
}

func IsPolicyValueEqual(v1, v2 TDnsPolicyTypeValue) bool {
	return jsonutils.Marshal(v1).Equals(jsonutils.Marshal(v2))
}

type DnsRecordSet struct {
	Id         string
	ExternalId string

	Enabled      bool
	DnsName      string
	DnsType      TDnsType
	DnsValue     string
	Status       string
	Ttl          int64
	PolicyType   TDnsPolicyType
	PolicyParams TDnsPolicyTypeValue
}

func (r DnsRecordSet) GetGlobalId() string {
	return r.ExternalId
}

func (r DnsRecordSet) GetName() string {
	return r.DnsName
}

func (r DnsRecordSet) GetDnsName() string {
	return r.DnsName
}

func (r DnsRecordSet) GetDnsValue() string {
	return r.DnsValue
}

func (r DnsRecordSet) GetPolicyType() TDnsPolicyType {
	return r.PolicyType
}

func (r DnsRecordSet) GetPolicyParams() TDnsPolicyTypeValue {
	return r.PolicyParams
}

func (r DnsRecordSet) GetStatus() string {
	return r.Status
}

func (r DnsRecordSet) GetTTL() int64 {
	return r.Ttl
}

func (r DnsRecordSet) GetDnsType() TDnsType {
	return r.DnsType
}

func (r DnsRecordSet) GetEnabled() bool {
	return r.Enabled
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
	if record.Enabled != r.Enabled {
		return false
	}
	if !IsPolicyValueEqual(record.PolicyParams, r.PolicyParams) {
		return false
	}
	return true
}

func (record DnsRecordSet) String() string {
	if record.PolicyParams != nil {
		return fmt.Sprintf("%s-%s-%s-%s-%s", record.DnsName, record.DnsType, record.DnsValue, record.PolicyType, record.PolicyParams.String())
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
	return strings.Compare(records[i].String(), records[j].String()) < 0
}

func CompareDnsRecordSet(iRecords []ICloudDnsRecordSet, locals []DnsRecordSet, debug bool) ([]DnsRecordSet, []DnsRecordSet, []DnsRecordSet, []DnsRecordSet) {
	remote := map[TDnsType]DnsRecordSets{}
	local := map[TDnsType]DnsRecordSets{}
	for i := range iRecords {
		record := DnsRecordSet{
			ExternalId: iRecords[i].GetGlobalId(),

			DnsName:      iRecords[i].GetDnsName(),
			DnsType:      iRecords[i].GetDnsType(),
			DnsValue:     iRecords[i].GetDnsValue(),
			Status:       iRecords[i].GetStatus(),
			Enabled:      iRecords[i].GetEnabled(),
			Ttl:          iRecords[i].GetTTL(),
			PolicyType:   iRecords[i].GetPolicyType(),
			PolicyParams: iRecords[i].GetPolicyParams(),
		}
		_, ok := remote[record.DnsType]
		if !ok {
			remote[record.DnsType] = DnsRecordSets{}
		}
		remote[record.DnsType] = append(remote[record.DnsType], record)
	}
	for i := range locals {
		_, ok := local[locals[i].DnsType]
		if !ok {
			local[locals[i].DnsType] = DnsRecordSets{}
		}
		local[locals[i].DnsType] = append(local[locals[i].DnsType], locals[i])
	}

	var compare = func(remote, local DnsRecordSets) ([]DnsRecordSet, []DnsRecordSet, []DnsRecordSet, []DnsRecordSet) {
		common, add, del, update := []DnsRecordSet{}, []DnsRecordSet{}, []DnsRecordSet{}, []DnsRecordSet{}
		i, j := 0, 0
		for i < len(local) || j < len(remote) {
			if i < len(local) && j < len(remote) {
				l, r := local[i].String(), remote[j].String()
				if debug {
					log.Debugf("compare %s -> %s", l, r)
				}
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

	common, add, del, update := []DnsRecordSet{}, []DnsRecordSet{}, []DnsRecordSet{}, []DnsRecordSet{}
	for _, dnsType := range SUPPORTED_DNS_TYPES {
		_remote, _ := remote[dnsType]
		_local, _ := local[dnsType]
		sort.Sort(_remote)
		sort.Sort(_local)
		_common, _add, _del, _update := compare(_remote, _local)
		common = append(common, _common...)
		add = append(add, _add...)
		del = append(del, _del...)
		update = append(update, _update...)
	}
	return common, add, del, update
}
