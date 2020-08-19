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

func (client *SAwsClient) associateVPCWithHostedZone(vpcId string, regionId string, hostedZoneId string) error {
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

func (client *SAwsClient) disassociateVPCFromHostedZone(vpcId string, regionId string, hostedZoneId string) error {
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

// Create, Delete, and Upsert
func (client *SAwsClient) changeResourceRecordSets(opts *cloudprovider.SDnsRecordSetChangeOptions, action string, hostedZoneId string) error {
	s, err := client.getAwsRoute53Session()
	if err != nil {
		return errors.Wrap(err, "region.getAwsRoute53Session()")
	}
	// prebuild recordset params
	resourceRecordSet := route53.ResourceRecordSet{}
	resourceRecordSet.Name = &opts.Name
	values := strings.Split(opts.Value, "*")
	resourceRecords := []*route53.ResourceRecord{}
	for i := 0; i < len(values); i++ {
		resourceRecords = append(resourceRecords, &route53.ResourceRecord{Value: &values[i]})
	}
	resourceRecordSet.ResourceRecords = resourceRecords

	route53Client := route53.New(s)
	params := route53.ChangeResourceRecordSetsInput{}
	ChangeBatch := route53.ChangeBatch{}
	change := route53.Change{}
	change.Action = &action
	change.ResourceRecordSet = &resourceRecordSet

	ChangeBatch.Changes = []*route53.Change{&change}
	params.HostedZoneId = &hostedZoneId
	//params.ChangeBatch=
	_, err = route53Client.ChangeResourceRecordSets(&params)
	if err != nil {
		return errors.Wrap(err, "route53Client.ChangeResourceRecordSets(&params)")
	}
	return nil
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

func (self *ShostedZone) AddVpc(vpcId string, vpcRegionId string) error {
	if self.Config.PrivateZone {
		err := self.client.associateVPCWithHostedZone(vpcId, vpcRegionId, self.ID)
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
		err := self.client.disassociateVPCFromHostedZone(vpcId, vpcRegionId, self.ID)
		if err != nil {
			return errors.Wrapf(err, "self.client.disassociateVPCFromHostedZone(%s,%s,%s)", vpcId, vpcRegionId, self.ID)
		}
	} else {
		return errors.Wrap(cloudprovider.ErrNotSupported, "public hostedZone not support disassociate vpc")
	}
	return nil
}

func (self *ShostedZone) GetIDnsRecordSets() ([]cloudprovider.ICloudDnsRecordSet, error) {
	recordSets, err := self.client.GetSdnsRecordSets(self.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "self.client.GetSdnsRecordSets(%s)", self.ID)
	}
	result := []cloudprovider.ICloudDnsRecordSet{}
	for i := 0; i < len(recordSets); i++ {
		result = append(result, &recordSets[i])
	}
	return result, nil
}
func (self *ShostedZone) AddDnsRecordSet(opts *cloudprovider.SAddDnsRecordSetOptions) error {
	err := self.client.changeResourceRecordSets(&opts.SDnsRecordSetChangeOptions, "Create", self.ID)
	if err != nil {
		return errors.Wrapf(err, `self.client.changeResourceRecordSets(opts, "Create",%s)`, self.ID)
	}
	return nil
}
func (self *ShostedZone) RemoveDnsRecordSet(opts *cloudprovider.SRemoveDnsRecordSetOptions) error {
	err := self.client.changeResourceRecordSets(&opts.SDnsRecordSetChangeOptions, "Delete", self.ID)
	if err != nil {
		return errors.Wrapf(err, `self.client.changeResourceRecordSets(opts, "Create",%s)`, self.ID)
	}
	return nil
}
