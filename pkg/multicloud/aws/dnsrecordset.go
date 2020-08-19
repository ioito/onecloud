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
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/service/route53"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
)

type resourceRecord struct {
	Value string `json:"Value"`
}

type SGeoLocationCode struct {
	// The two-letter code for the continent.
	//
	// Valid values: AF | AN | AS | EU | OC | NA | SA
	//
	// Constraint: Specifying ContinentCode with either CountryCode or SubdivisionCode
	// returns an InvalidInput error.
	ContinentCode string `json:"ContinentCode"`

	// The two-letter code for the country.
	CountryCode string `json:"CountryCode"`

	// The code for the subdivision. Route 53 currently supports only states in
	// the United States.
	SubdivisionCode string `json:"SubdivisionCode"`
}

type SAliasTarget struct {
	DNSName              string `json:"DNSName"`
	EvaluateTargetHealth *bool  `json:"EvaluateTargetHealth"`
	HostedZoneId         string `json:"HostedZoneId"`
}
type SdnsRecordSet struct {
	client                  *SAwsClient
	AliasTarget             SAliasTarget `json:"AliasTarget"`
	hostedzoneID            string
	Name                    string           `json:"Name"`
	ResourceRecords         []resourceRecord `json:"ResourceRecords"`
	TTL                     string           `json:"TTL"`
	TrafficPolicyInstanceId string           `json:"TrafficPolicyInstanceId"`
	Type                    string           `json:"Type"`
	SetIdentifier           string           `json:"SetIdentifier"` // 区别 多值 等名称重复的记录
	// policy info
	Failover         string            `json:"Failover"`
	GeoLocation      *SGeoLocationCode `json:"GeoLocation"`
	Region           string            `json:"Region"` // latency based
	MultiValueAnswer *bool             `json:"MultiValueAnswer"`
	Weight           *int64            `json:"Weight"`

	HealthCheckId string `json:"HealthCheckId"`
	policyinfo    *jsonutils.JSONDict
}

type RecordSetChange struct {
}

func (client *SAwsClient) GetSdnsRecordSets(HostedZoneId string) ([]SdnsRecordSet, error) {
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return nil, errors.Wrap(err, "region.getAwsRoute53Session()")
	}
	route53Client := route53.New(s)
	result := []SdnsRecordSet{}
	params := route53.ListResourceRecordSetsInput{}
	StartRecordName := ""
	MaxItems := "100"

	for true {
		if len(StartRecordName) > 0 {
			params.StartRecordName = &StartRecordName
		}
		params.MaxItems = &MaxItems
		params.HostedZoneId = &HostedZoneId
		ret, err := route53Client.ListResourceRecordSets(&params)
		if err != nil {
			return nil, errors.Wrap(err, "route53Client.ListResourceRecordSets()")
		}

		recordsSets := []SdnsRecordSet{}
		err = unmarshalAwsOutput(ret, "ResourceRecordSets", &recordsSets)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshalAwsOutput(ResourceRecordSets)")
		}
		result = append(result, recordsSets...)
		if !*ret.IsTruncated {
			break
		}
		StartRecordName = *ret.NextRecordName
	}
	for i := 0; i < len(result); i++ {
		result[i].hostedzoneID = HostedZoneId
	}

	return result, nil
}

func (self *SdnsRecordSet) GetGlobalId() string {
	return ""
}
func (self *SdnsRecordSet) GetDnsName() string {
	return self.Name
}
func (self *SdnsRecordSet) GetDnsType() string {
	return self.Type
}
func (self *SdnsRecordSet) GetDnsValue() string {
	var records []string
	for i := 0; i < len(self.ResourceRecords); i++ {
		records = append(records, self.ResourceRecords[i].Value)
	}
	return strings.Join(records, "*")
}
func (self *SdnsRecordSet) GetTTL() int {
	i, err := strconv.Atoi(self.TTL)
	if err != nil {
		return 0
	}
	return i
}

func (self *SdnsRecordSet) GetICloudVpcIds() ([]string, error) {
	return nil, nil
}
func (self *SdnsRecordSet) GetICloudDnsTrafficPolicy() (cloudprovider.ICloudDnsTrafficPolicy, error) {
	return self, nil
}
func (self *SdnsRecordSet) SetICloudDnsTrafficePolicy(opts *cloudprovider.SDnsTrafficPolicySetOptions) error {
	return nil
}

// trafficpolicy 信息
func (self *SdnsRecordSet) GetPolicyType() cloudprovider.TDnsPolicyType {
	/*
		Failover         string          `json:"Failover"`
		GeoLocation      GeoLocationCode `json:"GeoLocation"`
		Region           string          `json:"Region"` // latency based
		MultiValueAnswer *bool           `json:"MultiValueAnswer"`
		Weight           *int64          `json:"Weight"`
	*/
	self.policyinfo = jsonutils.NewDict()
	if len(self.Failover) > 0 {
		self.policyinfo.Add(jsonutils.Marshal(self.Failover), "Failover")
		self.policyinfo.Add(jsonutils.Marshal(self.HealthCheckId), "HealthCheckId")
		return cloudprovider.DnsPolicyTypeFailover
	}
	if self.GeoLocation != nil {
		self.policyinfo.Add(jsonutils.Marshal(self.GeoLocation), "GeoLocation")
		return cloudprovider.DnsPolicyTypeByGeoLocation
	}
	if len(self.Region) > 0 {
		self.policyinfo.Add(jsonutils.Marshal(self.Region), "Region")
		return cloudprovider.DnsPolicyTypeLatency
	}
	if self.MultiValueAnswer != nil {
		self.policyinfo.Add(jsonutils.Marshal(self.MultiValueAnswer), "MultiValueAnswer")
		return cloudprovider.DnsPolicyTypeMultiValueAnswer
	}
	if self.Weight != nil {
		self.policyinfo.Add(jsonutils.Marshal(self.Weight), "Weight")
		return cloudprovider.DnsPolicyTypeWeighted
	}
	return cloudprovider.DnsPolicyTypeSimple

}
func (self *SdnsRecordSet) GetParams() *jsonutils.JSONDict {
	self.GetPolicyType()
	return self.policyinfo
}
