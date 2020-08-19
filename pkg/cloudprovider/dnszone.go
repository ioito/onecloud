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

type TDnsZoneType string
type TDnsPolicyType string

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
	DnsPolicyTypeFailover         = TDnsPolicyType("Failover")         //failover
	DnsPolicyTypeLatency          = TDnsPolicyType("Latency")          //faulover
	DnsPolicyTypeMultiValueAnswer = TDnsPolicyType("MultiValueAnswer") //faulover

)

type SDnsZoneCreateOptions struct {
}

type SDnsTrafficPolicySetOptions struct {
}

type SAddDnsRecordSetOptions struct {
	SDnsRecordSetChangeOptions
}

type SRemoveDnsRecordSetOptions struct {
	SDnsRecordSetChangeOptions
}

type SDnsRecordSetChangeOptions struct {
	Name  string
	Value string //joined by '*'
	TTL   int64
	Type  string
}
