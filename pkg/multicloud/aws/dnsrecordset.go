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
	"strings"

	"github.com/aws/aws-sdk-go/service/route53"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
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
	hostedZone              *SHostedZone
	AliasTarget             SAliasTarget     `json:"AliasTarget"`
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
}

func (client *SAwsClient) GetSdnsRecordSets(HostedZoneId string) ([]SdnsRecordSet, error) {
	resourceRecordSets, err := client.GetRoute53ResourceRecordSets(HostedZoneId)
	if err != nil {
		return nil, errors.Wrapf(err, "client.GetRoute53ResourceRecordSets(%s)", HostedZoneId)
	}
	result := []SdnsRecordSet{}
	err = unmarshalAwsOutput(resourceRecordSets, "", &result)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalAwsOutput(ResourceRecordSets)")
	}

	return result, nil
}

func (client *SAwsClient) GetRoute53ResourceRecordSets(HostedZoneId string) ([]*route53.ResourceRecordSet, error) {
	// client
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return nil, errors.Wrap(err, "client.getAwsRoute53Session()")
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
		if ret.IsTruncated == nil || !*ret.IsTruncated {
			break
		}
		StartRecordName = *ret.NextRecordName
	}
	return resourceRecordSets, nil
}

// CREATE, DELETE, UPSERT
func (client *SAwsClient) ChangeResourceRecordSets(action string, hostedZoneId string, resourceRecordSets ...*route53.ResourceRecordSet) error {
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return errors.Wrap(err, "client.getAwsRoute53Session()")
	}
	route53Client := route53.New(s)

	ChangeBatch := route53.ChangeBatch{}
	for i := 0; i < len(resourceRecordSets); i++ {
		change := route53.Change{}
		change.Action = &action
		change.ResourceRecordSet = resourceRecordSets[i]
		ChangeBatch.Changes = append(ChangeBatch.Changes, &change)
	}

	changeParams := route53.ChangeResourceRecordSetsInput{}
	changeParams.HostedZoneId = &hostedZoneId
	changeParams.ChangeBatch = &ChangeBatch
	_, err = route53Client.ChangeResourceRecordSets(&changeParams)
	if err != nil {
		return errors.Wrap(err, "route53Client.ChangeResourceRecordSets(&params)")
	}
	return nil
}

func Getroute53ResourceRecordSet(opts *cloudprovider.DnsRecordSet) (*route53.ResourceRecordSet, error) {
	resourceRecordSet := route53.ResourceRecordSet{}
	resourceRecordSet.SetName(opts.DnsName)
	resourceRecordSet.SetTTL(opts.Ttl)
	resourceRecordSet.SetType(string(opts.DnsType))
	if len(opts.ExternalId) > 0 {
		resourceRecordSet.SetSetIdentifier(opts.ExternalId)
	}
	records := []*route53.ResourceRecord{}
	values := strings.Split(opts.DnsValue, " ")
	for i := 0; i < len(values); i++ {
		records = append(records, &route53.ResourceRecord{Value: &values[i]})
	}
	resourceRecordSet.SetResourceRecords(records)

	// traffic policy info--------------------------------------------
	if opts.PolicyType == cloudprovider.DnsPolicyTypeSimple || opts.PolicyParams == nil {
		return &resourceRecordSet, nil
	}

	var healthCheckId string
	err := opts.PolicyParams.Unmarshal(&healthCheckId, "healthcheckid")
	if err != nil {
		return nil, errors.Wrapf(err, "%s Unmarshal(healthcheckid)", fmt.Sprintln(opts.PolicyParams))
	}
	if len(healthCheckId) > 0 {
		resourceRecordSet.SetHealthCheckId(healthCheckId)
	}

	if opts.PolicyType == cloudprovider.DnsPolicyTypeFailover {
		var failover string
		err := opts.PolicyParams.Unmarshal(&failover, "failover")
		if err != nil {
			return nil, errors.Wrapf(err, "%s Unmarshal(failover)", fmt.Sprintln(opts.PolicyParams))
		}
		resourceRecordSet.SetFailover(failover)
	}
	if opts.PolicyType == cloudprovider.DnsPolicyTypeByGeoLocation {
		sGeo := SGeoLocationCode{}
		err := opts.PolicyParams.Unmarshal(&sGeo, "location")
		if err != nil {
			return nil, errors.Wrapf(err, "%s Unmarshal(location)", fmt.Sprintln(opts.PolicyParams))
		}
		Geo := route53.GeoLocation{}
		Geo.ContinentCode = &sGeo.ContinentCode
		Geo.CountryCode = &sGeo.CountryCode
		Geo.SubdivisionCode = &sGeo.SubdivisionCode
		resourceRecordSet.SetGeoLocation(&Geo)
	}

	if opts.PolicyType == cloudprovider.DnsPolicyTypeLatency {
		var region string
		err := opts.PolicyParams.Unmarshal(&region, "region")
		if err != nil {
			return nil, errors.Wrapf(err, "%s Unmarshal(region)", fmt.Sprintln(opts.PolicyParams))
		}
		resourceRecordSet.SetRegion(region)
	}
	if opts.PolicyType == cloudprovider.DnsPolicyTypeMultiValueAnswer {
		var multiValueAnswer bool
		err := opts.PolicyParams.Unmarshal(&multiValueAnswer, "multivalueanswer")
		if err != nil {
			return nil, errors.Wrapf(err, "%s Unmarshal(multivalueanswer)", fmt.Sprintln(opts.PolicyParams))
		}
		resourceRecordSet.SetMultiValueAnswer(multiValueAnswer)
	}
	if opts.PolicyType == cloudprovider.DnsPolicyTypeWeighted {
		var Weight int64
		err := opts.PolicyParams.Unmarshal(&Weight, "weight")
		if err != nil {
			return nil, errors.Wrapf(err, "%s Unmarshal(weight)", fmt.Sprintln(opts.PolicyParams))
		}
	}
	return &resourceRecordSet, nil

}

func (client *SAwsClient) AddDnsRecordSet(hostedZoneId string, opts *cloudprovider.DnsRecordSet) error {
	resourceRecordSet, err := Getroute53ResourceRecordSet(opts)
	if err != nil {
		return errors.Wrapf(err, "Getroute53ResourceRecordSet(%s)", fmt.Sprintln(opts))
	}
	err = client.ChangeResourceRecordSets("CREATE", hostedZoneId, resourceRecordSet)
	if err != nil {
		return errors.Wrapf(err, `self.client.changeResourceRecordSets(opts, "CREATE",%s)`, hostedZoneId)
	}
	return nil
}

func (client *SAwsClient) UpdateDnsRecordSet(hostedZoneId string, opts *cloudprovider.DnsRecordSet) error {
	resourceRecordSet, err := Getroute53ResourceRecordSet(opts)
	if err != nil {
		return errors.Wrapf(err, "Getroute53ResourceRecordSet(%s)", fmt.Sprintln(opts))
	}
	err = client.ChangeResourceRecordSets("UPSERT", hostedZoneId, resourceRecordSet)
	if err != nil {
		return errors.Wrapf(err, `self.client.changeResourceRecordSets(opts, "CREATE",%s)`, hostedZoneId)
	}
	return nil
}

func (client *SAwsClient) RemoveDnsRecordSet(hostedZoneId string, opts *cloudprovider.DnsRecordSet) error {
	resourceRecordSets, err := client.GetRoute53ResourceRecordSets(hostedZoneId)
	if err != nil {
		return errors.Wrapf(err, "self.client.GetRoute53ResourceRecordSets(%s)", hostedZoneId)
	}
	for i := 0; i < len(resourceRecordSets); i++ {
		srecordSet := SdnsRecordSet{}
		err = unmarshalAwsOutput(resourceRecordSets[i], "", &srecordSet)
		if err != nil {
			return errors.Wrap(err, "unmarshalAwsOutput(ResourceRecordSets)")
		}
		if srecordSet.match(opts) {
			err := client.ChangeResourceRecordSets("DELETE", hostedZoneId, resourceRecordSets[i])
			if err != nil {
				return errors.Wrapf(err, `self.client.changeResourceRecordSets(opts, "DELETE",%s)`, hostedZoneId)
			}
			return nil
		}
	}
	return nil
}

func (self *SdnsRecordSet) GetStatus() string {
	return ""
}

func (self *SdnsRecordSet) GetEnabled() bool {
	return true
}

func (self *SdnsRecordSet) GetGlobalId() string {
	return self.SetIdentifier
}

func (self *SdnsRecordSet) GetDnsName() string {
	if self.hostedZone == nil {
		return self.Name
	}
	if self.Name == self.hostedZone.Name {
		return "@"
	}
	return strings.TrimSuffix(self.Name, "."+self.hostedZone.Name)
}

func (self *SdnsRecordSet) GetDnsType() cloudprovider.TDnsType {
	return cloudprovider.TDnsType(self.Type)
}

func (self *SdnsRecordSet) GetDnsValue() string {
	var records []string
	for i := 0; i < len(self.ResourceRecords); i++ {
		records = append(records, self.ResourceRecords[i].Value)
	}
	return strings.Join(records, " ")
}

func (self *SdnsRecordSet) GetTTL() int64 {
	return self.TTL
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

	if len(self.Failover) > 0 {
		return cloudprovider.DnsPolicyTypeFailover
	}
	if self.GeoLocation != nil {
		return cloudprovider.DnsPolicyTypeByGeoLocation
	}
	if len(self.Region) > 0 {
		return cloudprovider.DnsPolicyTypeLatency
	}
	if self.MultiValueAnswer != nil {
		return cloudprovider.DnsPolicyTypeMultiValueAnswer
	}
	if self.Weight != nil {
		return cloudprovider.DnsPolicyTypeWeighted
	}
	return cloudprovider.DnsPolicyTypeSimple

}

func (self *SdnsRecordSet) GetPolicyParams() cloudprovider.TDnsPolicyTypeValue {
	policyinfo := jsonutils.NewDict()
	if len(self.HealthCheckId) > 0 {
		policyinfo.Add(jsonutils.Marshal(self.HealthCheckId), "healthcheckid")
	}
	if len(self.Failover) > 0 {
		policyinfo.Add(jsonutils.Marshal(self.Failover), "failover")
	}
	if self.GeoLocation != nil {
		policyinfo.Add(jsonutils.Marshal(self.GeoLocation), "location")
	}
	if len(self.Region) > 0 {
		policyinfo.Add(jsonutils.Marshal(self.Region), "region")
	}
	if self.MultiValueAnswer != nil {
		policyinfo.Add(jsonutils.Marshal(self.MultiValueAnswer), "multivalueanswer")
	}
	if self.Weight != nil {
		policyinfo.Add(jsonutils.Marshal(self.Weight), "weight")
	}
	return policyinfo
}

func (self *SdnsRecordSet) match(change *cloudprovider.DnsRecordSet) bool {
	if change.DnsName != self.GetDnsName() {
		return false
	}
	if change.DnsValue != self.GetDnsValue() {
		return false
	}
	if change.Ttl != self.GetTTL() {
		return false
	}
	if change.DnsType != self.GetDnsType() {
		return false
	}
	if change.ExternalId != self.GetGlobalId() {
		return false
	}
	return true
}
