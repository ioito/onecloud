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
	"time"

	"github.com/aws/aws-sdk-go/service/route53"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
)

type HostedZoneConfig struct {
	Comment     string `json:"Comment"`
	PrivateZone bool   `json:"PrivateZone"`
}

type AssociatedVPC struct {
	VPCId     string `json:"VPCId"`
	VPCRegion string `json:"VPCRegion"`
}
type ShostedZone struct {
	client                 *SAwsClient
	recordSets             []SdnsRecordSet
	ID                     string           `json:"Id"`
	Name                   string           `json:"Name"`
	Config                 HostedZoneConfig `json:"Config"`
	ResourceRecordSetCount int64            `json:"ResourceRecordSetCount"`
	VPCs                   []AssociatedVPC  `json:"VPCs"`
}

func (self *ShostedZone) GetId() string {
	return self.ID
}
func (self *ShostedZone) GetName() string {
	return self.Name
}
func (self *ShostedZone) GetGlobalId() string {
	return self.ID
}

func (self *ShostedZone) GetStatus() string {
	return ""
}

func (self *ShostedZone) Refresh() error {
	return nil
}

func (self *ShostedZone) IsEmulated() bool {
	return false
}
func (self *ShostedZone) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (client *SAwsClient) CreateHostedZone(opts *cloudprovider.SDnsZoneCreateOptions) (*ShostedZone, error) {
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return nil, errors.Wrap(err, "region.getAwsRoute53Session()")
	}
	route53Client := route53.New(s)
	params := route53.CreateHostedZoneInput{}
	timeStirng := time.Now().String()
	params.CallerReference = &timeStirng
	params.Name = &opts.Name

	Config := route53.HostedZoneConfig{}
	vpc := route53.VPC{}
	var IsPrivate bool

	if opts.ZoneType == cloudprovider.PrivateZone {
		IsPrivate = true
		vpc.VPCId = &opts.VpcId
		vpc.VPCRegion = &opts.VpcRegionId
		params.VPC = &vpc
	}
	Config.Comment = &opts.Desc
	Config.PrivateZone = &IsPrivate

	ret, err := route53Client.CreateHostedZone(&params)
	if err != nil {
		return nil, errors.Wrap(err, "route53Client.GetHostedZone()")
	}
	result := ShostedZone{}
	err = unmarshalAwsOutput(ret, "HostedZone", &result)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalAwsOutput(HostedZone)")
	}
	return &result, nil
}

func (client *SAwsClient) DeleteHostedZone(Id string) error {
	// client
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return errors.Wrap(err, "region.getAwsRoute53Session()")
	}
	route53Client := route53.New(s)

	// fetch records
	resourceRecordSets, err := client.GetRoute53ResourceRecordSets(Id)
	if err != nil {
		return errors.Wrapf(err, "client.GetRoute53ResourceRecordSets(%s)", Id)
	}
	// prepare batch and delete
	deleteRecordSets := []*route53.ResourceRecordSet{}
	for i := 0; i < len(resourceRecordSets); i++ {
		var dnsType string
		if resourceRecordSets[i].Type != nil {
			dnsType = *resourceRecordSets[i].Type
		}
		if dnsType == "NS" || dnsType == "SOA" {
			continue
		}
		deleteRecordSets = append(deleteRecordSets, resourceRecordSets[i])
	}
	err = client.ChangeResourceRecordSets("DELETE", Id, deleteRecordSets...)
	if err != nil {
		return errors.Wrapf(err, "client.ChangeResourceRecordSets(DELETE, %s, deleteRecordSets)", Id)
	}

	// delete hostedzone
	params := route53.DeleteHostedZoneInput{}
	params.Id = &Id
	_, err = route53Client.DeleteHostedZone(&params)
	if err != nil {
		return errors.Wrapf(err, "route53Client.DeleteHostedZone(%s)", Id)
	}
	return nil
}

func (client *SAwsClient) CreateICloudDnsZone(opts *cloudprovider.SDnsZoneCreateOptions) (cloudprovider.ICloudDnsZone, error) {
	return client.CreateHostedZone(opts)
}

func (client *SAwsClient) GetHostedZones() ([]ShostedZone, error) {
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return nil, errors.Wrap(err, "region.getAwsRoute53Session()")
	}
	route53Client := route53.New(s)
	result := []ShostedZone{}
	Marker := ""
	MaxItems := "100"
	params := route53.ListHostedZonesInput{}
	for true {
		if len(Marker) > 0 {
			params.Marker = &Marker
		}
		params.MaxItems = &MaxItems
		ret, err := route53Client.ListHostedZones(&params)
		if err != nil {
			return nil, errors.Wrap(err, "route53Client.ListHostedZones(nil)")
		}
		hostedZones := []ShostedZone{}
		err = unmarshalAwsOutput(ret, "HostedZones", &hostedZones)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshalAwsOutput(HostedZones)")
		}
		result = append(result, hostedZones...)
		if !*ret.IsTruncated {
			break
		}
		if ret.Marker != nil {
			Marker = *ret.Marker
		}

	}

	return result, nil
}

func (client *SAwsClient) GetICloudDnsZones() ([]cloudprovider.ICloudDnsZone, error) {
	hostedZones, err := client.GetHostedZones()
	if err != nil {
		return nil, errors.Wrap(err, "client.GetHostedZones()")
	}
	result := []cloudprovider.ICloudDnsZone{}
	for i := 0; i < len(hostedZones); i++ {
		result = append(result, &hostedZones[i])
	}
	return result, nil
}

func (client *SAwsClient) GetHostedZoneById(ID string) (*ShostedZone, error) {
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return nil, errors.Wrap(err, "region.getAwsRoute53Session()")
	}
	route53Client := route53.New(s)
	params := route53.GetHostedZoneInput{}
	params.Id = &ID
	ret, err := route53Client.GetHostedZone(&params)
	if err != nil {
		return nil, errors.Wrap(err, "route53Client.GetHostedZone()")
	}

	result := ShostedZone{}
	err = unmarshalAwsOutput(ret, "HostedZone", &result)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalAwsOutput(HostedZone)")
	}
	return &result, nil
}

func (client *SAwsClient) AssociateVPCWithHostedZone(vpcId string, regionId string, hostedZoneId string) error {
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return errors.Wrap(err, "region.getAwsRoute53Session()")
	}
	route53Client := route53.New(s)
	params := route53.AssociateVPCWithHostedZoneInput{}
	vpcParams := route53.VPC{}
	vpcParams.VPCId = &vpcId
	vpcParams.VPCRegion = &regionId
	params.VPC = &vpcParams
	params.HostedZoneId = &hostedZoneId

	_, err = route53Client.AssociateVPCWithHostedZone(&params)
	if err != nil {
		return errors.Wrap(err, "route53Client.AssociateVPCWithHostedZone()")
	}
	return nil
}

func (client *SAwsClient) DisassociateVPCFromHostedZone(vpcId string, regionId string, hostedZoneId string) error {
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return errors.Wrap(err, "region.getAwsRoute53Session()")
	}
	route53Client := route53.New(s)
	params := route53.DisassociateVPCFromHostedZoneInput{}
	vpcParams := route53.VPC{}
	vpcParams.VPCId = &vpcId
	vpcParams.VPCRegion = &regionId
	params.VPC = &vpcParams
	params.HostedZoneId = &hostedZoneId

	_, err = route53Client.DisassociateVPCFromHostedZone(&params)
	if err != nil {
		return errors.Wrap(err, "route53Client.AssociateVPCWithHostedZone()")
	}
	return nil
}

// CREATE, DELETE, UPSERT
func (client *SAwsClient) ChangeResourceRecordSets(action string, hostedZoneId string, resourceRecordSets ...*route53.ResourceRecordSet) error {
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return errors.Wrap(err, "region.getAwsRoute53Session()")
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

func (client *SAwsClient) AddDnsRecordSet(hostedZoneId string, opts cloudprovider.SDnsRecordSetChangeOptions) error {
	resourceRecordSet := route53.ResourceRecordSet{}
	resourceRecordSet.SetName(opts.Name)
	resourceRecordSet.SetTTL(opts.TTL)
	resourceRecordSet.SetType(opts.Type)
	records := []*route53.ResourceRecord{}
	values := strings.Split(opts.Value, "*")
	for i := 0; i < len(values); i++ {
		records = append(records, &route53.ResourceRecord{Value: &values[i]})
	}
	resourceRecordSet.SetResourceRecords(records)

	err := client.ChangeResourceRecordSets("CREATE", hostedZoneId, &resourceRecordSet)
	if err != nil {
		return errors.Wrapf(err, `self.client.changeResourceRecordSets(opts, "CREATE",%s)`, hostedZoneId)
	}
	return nil
}

func (client *SAwsClient) RemoveDnsRecordSet(hostedZoneId string, opts cloudprovider.SDnsRecordSetChangeOptions) error {
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
	return cloudprovider.ErrNotFound
}

func (self *ShostedZone) GetZoneType() cloudprovider.TDnsZoneType {
	if self.Config.PrivateZone {
		return cloudprovider.PrivateZone
	}
	return cloudprovider.PublicZone
}

func (self *ShostedZone) GetOptions() *jsonutils.JSONDict {
	return nil
}

func (self *ShostedZone) GetICloudVpcIds() ([]string, error) {
	vpcs := []string{}
	if self.Config.PrivateZone {
		for i := 0; i < len(self.VPCs); i++ {
			vpcs = append(vpcs, self.VPCs[i].VPCId)
		}
		return vpcs, nil
	}
	return vpcs, errors.Wrapf(cloudprovider.ErrNotSupported, "not a private hostedzone")
}

func (self *ShostedZone) AddVpc(vpcId string, vpcRegionId string) error {
	if self.Config.PrivateZone {
		err := self.client.AssociateVPCWithHostedZone(vpcId, vpcRegionId, self.ID)
		if err != nil {
			return errors.Wrapf(err, "self.client.associateVPCWithHostedZone(%s,%s,%s)", vpcId, vpcRegionId, self.ID)
		}
	} else {
		return errors.Wrap(cloudprovider.ErrNotSupported, "public hostedZone not support associate vpc")
	}
	return nil
}
func (self *ShostedZone) RemoveVpc(vpcId string, vpcRegionId string) error {
	if self.Config.PrivateZone {
		err := self.client.DisassociateVPCFromHostedZone(vpcId, vpcRegionId, self.ID)
		if err != nil {
			return errors.Wrapf(err, "self.client.disassociateVPCFromHostedZone(%s,%s,%s)", vpcId, vpcRegionId, self.ID)
		}
	} else {
		return errors.Wrap(cloudprovider.ErrNotSupported, "public hostedZone not support disassociate vpc")
	}
	return nil
}

func (self *ShostedZone) fetchRecordSets() error {
	recordSets, err := self.client.GetSdnsRecordSets(self.ID)
	if err != nil {
		return errors.Wrapf(err, "self.client.GetSdnsRecordSets(%s)", self.ID)
	}
	self.recordSets = recordSets
	return nil
}

func (self *ShostedZone) GetIDnsRecordSets() ([]cloudprovider.ICloudDnsRecordSet, error) {
	if self.recordSets == nil {
		err := self.fetchRecordSets()
		if err != nil {
			return nil, errors.Wrap(err, "self.fetchResourceRecordSets()")
		}
	}
	result := []cloudprovider.ICloudDnsRecordSet{}
	for i := 0; i < len(self.recordSets); i++ {
		result = append(result, &self.recordSets[i])
	}
	return result, nil
}
func (self *ShostedZone) AddDnsRecordSet(opts *cloudprovider.SAddDnsRecordSetOptions) error {
	return self.client.AddDnsRecordSet(self.ID, opts.SDnsRecordSetChangeOptions)
}
func (self *ShostedZone) RemoveDnsRecordSet(opts *cloudprovider.SRemoveDnsRecordSetOptions) error {
	return self.client.RemoveDnsRecordSet(self.ID, opts.SDnsRecordSetChangeOptions)
}
