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

package qcloud

import (
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
)

type sDomianCountInfo struct {
	DomainTotal int `json:"domain_total"`
}

type SDomian struct {
	client *SQcloudClient

	ID               int    `json:"id"`
	Status           string `json:"status"`
	GroupID          string `json:"group_id"`
	SearchenginePush string `json:"searchengine_push"`
	IsMark           string `json:"is_mark"`
	TTL              string `json:"ttl"`
	CnameSpeedup     string `json:"cname_speedup"`
	Remark           string `json:"remark"`
	CreatedOn        string `json:"created_on"`
	UpdatedOn        string `json:"updated_on"`
	QProjectID       int    `json:"q_project_id"`
	Punycode         string `json:"punycode"`
	ExtStatus        string `json:"ext_status"`
	SrcFlag          string `json:"src_flag"`
	Name             string `json:"name"`
	Grade            string `json:"grade"`
	GradeTitle       string `json:"grade_title"`
	IsVip            string `json:"is_vip"`
	Owner            string `json:"owner"`
	Records          string `json:"records"`
	MinTTL           int    `json:"min_ttl"`
}

// https://cloud.tencent.com/document/product/302/8505
func (client *SQcloudClient) GetDomains() ([]SDomian, error) {
	total := 0
	result := []SDomian{}
	for true {
		params := map[string]string{}
		params["offset"] = strconv.Itoa(total)
		resp, err := client.cnsRequest("DomainList", params)
		if err != nil {
			return nil, errors.Wrapf(err, "client.cnsRequest(DomainList, %s)", fmt.Sprintln(params))
		}
		count := sDomianCountInfo{}
		err = resp.Unmarshal(&count, "info")
		if err != nil {
			return nil, errors.Wrapf(err, "%s.Unmarshal(info)", fmt.Sprintln(resp))
		}
		domains := []SDomian{}
		err = resp.Unmarshal(&domains, "domains")
		if err != nil {
			return nil, errors.Wrapf(err, "%s.Unmarshal(domains)", fmt.Sprintln(resp))
		}
		if count.DomainTotal <= total {
			break
		}
		total += len(domains)
		result = append(result, domains...)
	}
	for i := 0; i < len(result); i++ {
		result[i].client = client
	}
	return result, nil
}

func (client *SQcloudClient) GetICloudDnsZones() ([]cloudprovider.ICloudDnsZone, error) {
	result := []cloudprovider.ICloudDnsZone{}
	domains, err := client.GetDomains()
	if err != nil {
		return nil, errors.Wrap(err, "client.GetDomains()")
	}
	for i := 0; i < len(domains); i++ {
		result = append(result, &domains[i])
	}
	return result, nil
}

func (client *SQcloudClient) GetDomainById(domainId string) (*SDomian, error) {
	domains, err := client.GetDomains()
	if err != nil {
		return nil, errors.Wrap(err, "client.GetDomains()")
	}
	for i := 0; i < len(domains); i++ {
		if strconv.Itoa(domains[i].ID) == domainId {
			return &domains[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "can't find %s in %s", domainId, fmt.Sprintln(domains))
}

// https://cloud.tencent.com/document/product/302/8504
func (client *SQcloudClient) CreateDomian(domianName string) (*SDomian, error) {
	params := map[string]string{}
	params["domain"] = domianName
	_, err := client.cnsRequest("DomainCreate", params)
	if err != nil {
		return nil, errors.Wrapf(err, "client.cnsRequest(DomainCreate, %s)", fmt.Sprintln(params))
	}
	domains, err := client.GetDomains()
	if err != nil {
		return nil, errors.Wrap(err, "client.GetDomains()")
	}
	for i := 0; i < len(domains); i++ {
		if domains[i].Name == domianName {
			return &domains[i], nil
		}
	}
	return nil, errors.Wrap(cloudprovider.ErrNotFound, "domain not found after create")
}

func (client *SQcloudClient) CreateICloudDnsZone(opts *cloudprovider.SDnsZoneCreateOptions) (cloudprovider.ICloudDnsZone, error) {
	return client.CreateDomian(opts.Name)
}

// https://cloud.tencent.com/document/product/302/3873
func (client *SQcloudClient) DeleteDomian(domianName string) error {
	params := map[string]string{}
	params["domain"] = domianName
	_, err := client.cnsRequest("DomainDelete", params)
	if err != nil {
		return errors.Wrapf(err, "client.cnsRequest(DomainDelete, %s)", fmt.Sprintln(params))
	}
	return nil
}

func (self *SDomian) GetId() string {
	return strconv.Itoa(self.ID)
}

func (self *SDomian) GetName() string {
	return self.Name
}

func (self *SDomian) GetGlobalId() string {
	return strconv.Itoa(self.ID)
}

func (self *SDomian) GetStatus() string {
	return self.Status
}

func (self *SDomian) Refresh() error {
	domains, err := self.client.GetDomains()
	if err != nil {
		return errors.Wrap(err, "self.client.GetDomains()")
	}
	for i := 0; i < len(domains); i++ {
		if self.ID == domains[i].ID {
			return jsonutils.Update(self, &domains[i])
		}
	}
	return cloudprovider.ErrNotFound
}

func (self *SDomian) IsEmulated() bool {
	return false
}
func (self *SDomian) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SDomian) GetZoneType() cloudprovider.TDnsZoneType {
	return cloudprovider.PublicZone
}
func (self *SDomian) GetOptions() *jsonutils.JSONDict {
	return nil
}

func (self *SDomian) GetICloudVpcIds() ([]string, error) {
	return nil, nil
}
func (self *SDomian) AddVpc(vpcId string, vpcRegionId string) error {
	return cloudprovider.ErrNotSupported
}
func (self *SDomian) RemoveVpc(vpcId string, vpcRegionId string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SDomian) GetIDnsRecordSets() ([]cloudprovider.ICloudDnsRecordSet, error) {
	records, err := self.client.GetDnsRecords(self.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "self.client.GetDnsRecords(%s)", self.Name)
	}
	result := []cloudprovider.ICloudDnsRecordSet{}
	for i := 0; i < len(records); i++ {
		result = append(result, &records[i])
	}
	return result, nil
}

func (self *SDomian) AddDnsRecordSet(opts *cloudprovider.SAddDnsRecordSetOptions) error {
	values := strings.Split(opts.Value, "*")
	for i := 0; i < len(values); i++ {
		opts.Value = values[i]
		err := self.client.CreateDnsRecord(&opts.SDnsRecordSetChangeOptions, self.Name)
		if err != nil {
			return errors.Wrapf(err, "self.client.CreateDnsRecord(%s, %s)", fmt.Sprintln(opts), self.Name)
		}
	}
	return nil
}
func (self *SDomian) RemoveDnsRecordSet(opts *cloudprovider.SRemoveDnsRecordSetOptions) error {

	records, err := self.client.GetDnsRecords(self.GetName())
	if err != nil {
		return errors.Wrapf(err, "self.client.GetDnsRecords(%s)", self.GetName())
	}
	for i := 0; i < len(records); i++ {
		if records[i].match(&opts.SDnsRecordSetChangeOptions) {
			err := self.client.DeleteDnsRecord(records[i].ID, self.GetName())
			if err != nil {
				return errors.Wrapf(err, "self.client.RemoveDnsRecord(%s,%s)", records[i].ID, self.GetName())
			}
		}
	}
	return cloudprovider.ErrNotFound
}
