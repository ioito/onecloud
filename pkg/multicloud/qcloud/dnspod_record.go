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
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
)

type SRecordCountInfo struct {
	RecordTotal string `json:"record_total"`
	RecordsNum  string `json:"records_num"`
	SubDomains  string `json:"sub_domains"`
}

type SDnsRecord struct {
	client *SQcloudClient

	domainName string
	ID         int    `json:"id"`
	TTL        int    `json:"ttl"`
	Value      string `json:"value"`
	Enabled    int    `json:"enabled"`
	Status     string `json:"status"`
	UpdatedOn  string `json:"updated_on"`
	QProjectID int    `json:"q_project_id"`
	Name       string `json:"name"`
	Line       string `json:"line"`
	LineID     string `json:"line_id"`
	Type       string `json:"type"`
	Remark     string `json:"remark"`
	Mx         int    `json:"mx"`
	Hold       string `json:"hold"`
}

func (client *SQcloudClient) GetDnsRecords(sDomainName string) ([]SDnsRecord, error) {
	total := 0
	result := []SDnsRecord{}
	for true {
		params := map[string]string{}
		params["offset"] = strconv.Itoa(total)
		params["domain"] = sDomainName
		resp, err := client.cnsRequest("RecordList", params)
		if err != nil {
			return nil, errors.Wrapf(err, "client.cnsRequest(RecordList, %s)", fmt.Sprintln(params))
		}
		log.Infof("resp:%s", resp)
		count := SRecordCountInfo{}
		err = resp.Unmarshal(&count, "info")
		if err != nil {
			return nil, errors.Wrapf(err, "%s.Unmarshal(info)", fmt.Sprintln(resp))
		}
		records := []SDnsRecord{}
		err = resp.Unmarshal(&records, "records")
		if err != nil {
			return nil, errors.Wrapf(err, "%s.Unmarshal(records)", fmt.Sprintln(resp))
		}
		RecordTotal, err := strconv.Atoi(count.RecordTotal)
		if err != nil {
			return nil, errors.Wrapf(err, "strconv.Atoi(%s)", count.RecordTotal)
		}
		if RecordTotal <= total {
			break
		}
		total += len(records)
		result = append(result, records...)
	}
	for i := 0; i < len(result); i++ {
		result[i].client = client
		result[i].domainName = sDomainName
	}
	return result, nil
}

// https://cloud.tencent.com/document/api/302/8516
func (client *SQcloudClient) CreateDnsRecord(opts *cloudprovider.SDnsRecordSetChangeOptions, domainName string) error {
	params := map[string]string{}
	recordline := "默认"
	if opts.TTL < 600 {
		opts.TTL = 600
	}
	if opts.TTL > 604800 {
		opts.TTL = 604800
	}
	subDomain := strings.TrimSuffix(opts.Name, "."+domainName)
	if len(subDomain) < 1 {
		subDomain = "@"
	}
	params["domain"] = domainName
	params["subDomain"] = subDomain
	params["recordType"] = opts.Type
	params["ttl"] = strconv.FormatInt(opts.TTL, 10)
	params["value"] = opts.Value
	params["recordLine"] = recordline
	_, err := client.cnsRequest("RecordCreate", params)
	if err != nil {
		return errors.Wrapf(err, "client.cnsRequest(RecordCreate, %s)", fmt.Sprintln(params))
	}
	return nil
}

// https://cloud.tencent.com/document/api/302/8514
func (client *SQcloudClient) DeleteDnsRecord(recordId int, domainName string) error {
	params := map[string]string{}
	params["domain"] = domainName
	params["recordId"] = strconv.Itoa(recordId)
	_, err := client.cnsRequest("RecordDelete", params)
	if err != nil {
		return errors.Wrapf(err, "client.cnsRequest(RecordDelete, %s)", fmt.Sprintln(params))
	}
	return nil
}

func (self *SDnsRecord) GetDnsName() string {
	return self.Name + self.domainName
}
func (self *SDnsRecord) GetStatus() string {
	return self.Status
}
func (self *SDnsRecord) GetDnsType() string {
	return self.Type
}
func (self *SDnsRecord) GetDnsValue() string {
	return self.Value
}
func (self *SDnsRecord) GetTTL() int64 {
	return int64(self.TTL)
}

func (self *SDnsRecord) GetPolicyType() cloudprovider.TDnsPolicyType {
	return cloudprovider.DnsPolicyTypeSimple
}

func (self *SDnsRecord) GetPolicyParams() *jsonutils.JSONDict {
	return nil
}

func (self *SDnsRecord) GetDnsIdentify() string {
	return strconv.Itoa(self.ID)
}

func (self *SDnsRecord) match(change *cloudprovider.SDnsRecordSetChangeOptions) bool {
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
