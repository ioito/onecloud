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

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SRecordCountInfo struct {
	RecordTotal string `json:"record_total"`
	RecordsNum  string `json:"records_num"`
	SubDomains  string `json:"sub_domains"`
}

type SDnsRecord struct {
	client     *SQcloudClient
	policyinfo cloudprovider.TDnsPolicyTypeValue

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

// https://cloud.tencent.com/document/product/302/8517
func (client *SQcloudClient) GetDnsRecords(sDomainName string, offset int, limit int) ([]SDnsRecord, int, error) {

	params := map[string]string{}
	params["offset"] = strconv.Itoa(offset)
	params["length"] = strconv.Itoa(limit)
	params["domain"] = sDomainName
	resp, err := client.cnsRequest("RecordList", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "client.cnsRequest(RecordList, %s)", fmt.Sprintln(params))
	}
	log.Infof("resp:%s", resp)
	count := SRecordCountInfo{}
	err = resp.Unmarshal(&count, "info")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "%s.Unmarshal(info)", fmt.Sprintln(resp))
	}
	records := []SDnsRecord{}
	err = resp.Unmarshal(&records, "records")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "%s.Unmarshal(records)", fmt.Sprintln(resp))
	}
	RecordTotal, err := strconv.Atoi(count.RecordTotal)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "strconv.Atoi(%s)", count.RecordTotal)
	}

	for i := 0; i < len(records); i++ {
		records[i].client = client
		records[i].domainName = sDomainName
	}
	return records, RecordTotal, nil
}

func (client *SQcloudClient) GetAllDnsRecords(sDomainName string) ([]SDnsRecord, error) {
	count := 0
	result := []SDnsRecord{}
	for true {
		records, total, err := client.GetDnsRecords(sDomainName, count, 100)
		if err != nil {
			return nil, errors.Wrapf(err, "client.GetDnsRecords(%s,%d,%d)", sDomainName, count, 100)
		}

		result = append(result, records...)
		count += len(records)
		if total <= count {
			break
		}
	}
	for i := 0; i < len(result); i++ {
		result[i].client = client
		result[i].domainName = sDomainName
	}
	return result, nil
}

func GetRecordLineLineType(policyinfo cloudprovider.TDnsPolicyTypeValue) string {
	switch policyinfo {
	case cloudprovider.DnsPolicyTypeByGeoLocationMainland:
		return "境内"
	case cloudprovider.DnsPolicyTypeByGeoLocationOversea:
		return "境外"

	case cloudprovider.DnsPolicyTypeByCarrierTelecom:
		return "电信"
	case cloudprovider.DnsPolicyTypeByCarrierUnicom:
		return "联通"
	case cloudprovider.DnsPolicyTypeByCarrierChinaMobile:
		return "移动"
	case cloudprovider.DnsPolicyTypeByCarrierCernet:
		return "教育网"

	case cloudprovider.DnsPolicyTypeBySearchEngineBaidu:
		return "百度"
	case cloudprovider.DnsPolicyTypeBySearchEngineGoogle:
		return "谷歌"
	case cloudprovider.DnsPolicyTypeBySearchEngineYoudao:
		return "有道"
	case cloudprovider.DnsPolicyTypeBySearchEngineBing:
		return "必应"
	case cloudprovider.DnsPolicyTypeBySearchEngineSousou:
		return "搜搜"
	case cloudprovider.DnsPolicyTypeBySearchEngineSougou:
		return "搜狗"
	case cloudprovider.DnsPolicyTypeBySearchEngineQihu360:
		return "奇虎"
	default:
		return "默认"
	}
	return "默认"
}

// https://cloud.tencent.com/document/api/302/8516
func (client *SQcloudClient) CreateDnsRecord(opts *cloudprovider.DnsRecordSet, domainName string) error {
	params := map[string]string{}
	recordline := GetRecordLineLineType(opts.PolicyParams)
	if opts.Ttl < 600 {
		opts.Ttl = 600
	}
	if opts.Ttl > 604800 {
		opts.Ttl = 604800
	}
	if len(opts.DnsName) < 1 {
		opts.DnsName = "@"
	}
	params["domain"] = domainName
	params["subDomain"] = opts.DnsName
	params["recordType"] = string(opts.DnsType)
	params["ttl"] = strconv.FormatInt(opts.Ttl, 10)
	params["value"] = opts.DnsValue
	params["recordLine"] = recordline
	_, err := client.cnsRequest("RecordCreate", params)
	if err != nil {
		return errors.Wrapf(err, "client.cnsRequest(RecordCreate, %s)", fmt.Sprintln(params))
	}
	return nil
}

// https://cloud.tencent.com/document/product/302/8511
func (client *SQcloudClient) ModifyDnsRecord(opts *cloudprovider.DnsRecordSet, domainName string) error {
	params := map[string]string{}
	recordline := GetRecordLineLineType(opts.PolicyParams)
	if opts.Ttl < 600 {
		opts.Ttl = 600
	}
	if opts.Ttl > 604800 {
		opts.Ttl = 604800
	}
	subDomain := strings.TrimSuffix(opts.DnsName, "."+domainName)
	if len(subDomain) < 1 {
		subDomain = "@"
	}
	params["domain"] = domainName
	params["recordId"] = opts.ExternalId
	params["subDomain"] = subDomain
	params["recordType"] = string(opts.DnsType)
	params["ttl"] = strconv.FormatInt(opts.Ttl, 10)
	params["value"] = opts.DnsValue
	params["recordLine"] = recordline
	_, err := client.cnsRequest("RecordModify", params)
	if err != nil {
		return errors.Wrapf(err, "client.cnsRequest(RecordModify, %s)", fmt.Sprintln(params))
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

func (self *SDnsRecord) GetGlobalId() string {
	return strconv.Itoa(self.ID)
}

func (self *SDnsRecord) GetDnsName() string {
	return self.Name
}

func (self *SDnsRecord) GetStatus() string {
	switch self.Status {
	case "enable":
		return api.DNS_ZONE_STATUS_AVAILABLE
	case "pause":
		return api.DNS_ZONE_STATUS_AVAILABLE
	default:
		return api.DNS_ZONE_STATUS_UNKNOWN
	}
}

func (self *SDnsRecord) GetEnabled() bool {
	if self.Status == "enable" {
		return true
	}
	return false
}

func (self *SDnsRecord) GetDnsType() cloudprovider.TDnsType {
	return cloudprovider.TDnsType(self.Type)
}

func (self *SDnsRecord) GetDnsValue() string {
	return self.Value
}

func (self *SDnsRecord) GetTTL() int64 {
	return int64(self.TTL)
}

func (self *SDnsRecord) GetPolicyType() cloudprovider.TDnsPolicyType {
	var policyType cloudprovider.TDnsPolicyType
	switch self.Line {
	case "境内", "境外":
		policyType = cloudprovider.DnsPolicyTypeByGeoLocation
	case "电信", "联通", "移动", "教育网":
		policyType = cloudprovider.DnsPolicyTypeByCarrier
	case "百度", "谷歌", "有道", "必应", "搜搜", "搜狗", "奇虎":
		policyType = cloudprovider.DnsPolicyTypeBySearchEngine
	default:
		policyType = cloudprovider.DnsPolicyTypeSimple
	}
	return policyType
}

func (self *SDnsRecord) GetPolicyParams() cloudprovider.TDnsPolicyTypeValue {
	switch self.Line {
	case "境内":
		self.policyinfo = cloudprovider.DnsPolicyTypeByGeoLocationMainland
	case "境外":
		self.policyinfo = cloudprovider.DnsPolicyTypeByGeoLocationOversea

	case "电信":
		self.policyinfo = cloudprovider.DnsPolicyTypeByCarrierTelecom
	case "联通":
		self.policyinfo = cloudprovider.DnsPolicyTypeByCarrierUnicom
	case "移动":
		self.policyinfo = cloudprovider.DnsPolicyTypeByCarrierChinaMobile
	case "教育网":
		self.policyinfo = cloudprovider.DnsPolicyTypeByCarrierCernet

	case "百度":
		self.policyinfo = cloudprovider.DnsPolicyTypeBySearchEngineBaidu
	case "谷歌":
		self.policyinfo = cloudprovider.DnsPolicyTypeBySearchEngineGoogle
	case "有道":
		self.policyinfo = cloudprovider.DnsPolicyTypeBySearchEngineYoudao
	case "必应":
		self.policyinfo = cloudprovider.DnsPolicyTypeBySearchEngineBing
	case "搜搜":
		self.policyinfo = cloudprovider.DnsPolicyTypeBySearchEngineSousou
	case "搜狗":
		self.policyinfo = cloudprovider.DnsPolicyTypeBySearchEngineSougou
	case "奇虎":
		self.policyinfo = cloudprovider.DnsPolicyTypeBySearchEngineQihu360
	default:
	}
	return self.policyinfo
}

func (self *SDnsRecord) match(change *cloudprovider.DnsRecordSet) bool {
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
