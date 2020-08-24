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

package compute

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	DNS_ZONE_STATUS_AVAILABLE          = "available"
	DNS_ZONE_STATUS_CREATING           = "creating"
	DNS_ZONE_STATUS_CREATE_FAILE       = "create_failed"
	DNS_ZONE_STATUS_UNCACHING          = "uncaching"
	DNS_ZONE_STATUS_UNCACHE_FAILED     = "uncache_failed"
	DNS_ZONE_STATUS_CACHING            = "caching"
	DNS_ZONE_STATUS_CACHE_FAILED       = "cache_failed"
	DNS_ZONE_STATUS_ADD_VPCS           = "add_vpcs"
	DNS_ZONE_STATUS_ADD_VPCS_FAILED    = "add_vpcs_failed"
	DNS_ZONE_STATUS_REMOVE_VPCS        = "remove_vpcs"
	DNS_ZONE_STATUS_REMOVE_VPCS_FAILED = "remove_vpcs_failed"
	DNS_ZONE_STATUS_DISABLED           = "disabled"
	DNS_ZONE_STATUS_LOCKED             = "locked"
)

type DnsZoneCreateInput struct {
	apis.EnabledStatusInfrasResourceBaseCreateInput

	// 区域类型
	//
	//
	// | 类型			| 说明    |
	// |----------		|---------|
	// | PublicZone		| 公有    |
	// | PrivateZone	| 私有    |
	ZoneType string `json:"zone_type"`
	// 额外参数

	// VPC id列表, 仅在zone_type为PrivateZone时生效, vpc列表必须属于同一个账号
	VpcIds []string `json:"vpc_ids"`

	// 云账号Id, 仅在zone_type为PublicZone时生效, 若为空则不会在云上创建
	CloudaccountId string `json:"cloudaccount_id"`

	// 额外信息
	Options *jsonutils.JSONDict `json:"options"`
}

type DnsZoneDetails struct {
	apis.EnabledStatusInfrasResourceBaseDetails
}

type DnsZoneListInput struct {
	apis.EnabledStatusInfrasResourceBaseListInput

	// 区域类型
	//
	//
	// | 类型			| 说明    |
	// |----------		|---------|
	// | PublicZone		| 公有    |
	// | PrivateZone	| 私有    |
	ZoneType string `json:"zone_type"`
}

type DnsZoneSyncStatusInput struct {
}

type DnsZoneCacheInput struct {
	// 云账号Id
	//
	//
	// | 要求								|
	// |----------							|
	// | 1. dns zone 状态必须为available		|
	// | 2. dns zone zone_type 必须为PublicZone |
	// | 3. 指定云账号未在云上创建相应的 dns zone |
	CloudaccountId string
}

type DnsZoneUnacheInput struct {
	// 云账号Id
	CloudaccountId string
}

type DnsZoneAddVpcsInput struct {
	// VPC id列表
	VpcIds []string `json:"vpc_ids"`
}

type DnsZoneRemoveVpcsInput struct {
	// VPC id列表
	VpcIds []string `json:"vpc_ids"`
}
