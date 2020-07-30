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

package cloudid

import "yunion.io/x/onecloud/pkg/apis"

const (
	SAML_PROVIDER_STATUS_READY         = "ready"         // 正常
	SAML_PROVIDER_STATUS_CREATING      = "creating"      // 创建中
	SAML_PROVIDER_STATUS_CREAT_FAILED  = "create_failed" // 创建失败
	SAML_PROVIDER_STATUS_DELETING      = "deleting"      // 删除中
	SAML_PROVIDER_STATUS_DELETE_FAILED = "delete_failed" // 删除失败
	SAML_PROVIDER_STATUS_UNKNOWN       = "unknown"       // 未知
)

type SAMLProviderListInput struct {
	apis.StatusInfrasResourceBaseListInput
}

type SAMLProviderCreateInput struct {
	apis.StatusInfrasResourceBaseCreateInput

	// 云账号Id
	CloudaccountId string `json:"cloudaccount_id"`

	// swagger:ignore
	EntityId string `json:"entity_id"`

	// 元数据文档(需要base64加密)
	MetadataDocument string `json:"metadata_document"`
}

type SAMLProviderDetails struct {
	apis.StatusInfrasResourceBaseDetails
}
