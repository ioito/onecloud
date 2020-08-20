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
	TTL                     int64            `json:"TTL"`
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

func (client *SAwsClient) GetSdnsRecordSets(HostedZoneId string) ([]SdnsRecordSet, error) {
	// ----
	resourceRecordSets, err := client.GetRoute53ResourceRecordSets(HostedZoneId)
	if err != nil {
		return nil, errors.Wrapf(err, "client.GetRoute53ResourceRecordSets(%s)", HostedZoneId)
	}
	result := []SdnsRecordSet{}
	for i := 0; i < len(resourceRecordSets); i++ {
		err = unmarshalAwsOutput(resourceRecordSets, "", &result)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshalAwsOutput(ResourceRecordSets)")
		}
	}
	return result, nil
}

func (client *SAwsClient) GetRoute53ResourceRecordSets(HostedZoneId string) ([]*route53.ResourceRecordSet, error) {
	// client
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return nil, errors.Wrap(err, "region.getAwsRoute53Session()")
	}
	route53Client := route53.New(s)

	// fetch records
	resourceRecordSets := []*route53.ResourceRecordSet{}
	listParams := route53.ListResourceRecordSetsInput{}
	StartRecordName := ""
	MaxItems := "100"
	for true {
		if len(StartRecordName) > 0 {
			listParams.StartRecordName = &StartRecordName
		}
		listParams.MaxItems = &MaxItems
		listParams.HostedZoneId = &HostedZoneId
		ret, err := route53Client.ListResourceRecordSets(&listParams)
		if err != nil {
			return nil, errors.Wrap(err, "route53Client.ListResourceRecordSets()")
		}
		resourceRecordSets = append(resourceRecordSets, ret.ResourceRecordSets...)
		if !*ret.IsTruncated {
			break
		}
		StartRecordName = *ret.NextRecordName
	}
	return resourceRecordSets, nil
}

func (self *SdnsRecordSet) GetStatus() string {
	return ""
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
func (self *SdnsRecordSet) GetTTL() int64 {
	return self.TTL
}

func (self *SdnsRecordSet) GetDnsIdentify() string {
	return self.SetIdentifier
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

func (self *SdnsRecordSet) GetPolicyParams() *jsonutils.JSONDict {
	self.GetPolicyType()
	return self.policyinfo
}

func (self *SdnsRecordSet) match(change cloudprovider.SDnsRecordSetChangeOptions) bool {
	if change.Name != self.GetDnsName() {
		return false
	}
	if change.Value != self.GetDnsValue() {
		return false
	}
	if change.TTL != self.GetTTL() {
		return false
	}
	if change.Type != self.GetDnsType() {
		return false
	}
	if change.Identify != self.GetDnsIdentify() {
		return false
	}
	return true
}
