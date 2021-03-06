// Copyright 2018 JDCLOUD.COM
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
//
// NOTE: This class is auto generated by the jdcloud code generator program.

package models


type UpdateOneClickAlarmOption struct {

    /* 告警通知联系人
in: body (Optional) */
    Contacts []BaseContact `json:"contacts"`

    /* 是否开启一键报警，1打开；0-关闭。默认为0 (Optional) */
    Enabled int64 `json:"enabled"`

    /* 通知策略
in: body (Optional) */
    NoticeOption []NoticeOption `json:"noticeOption"`

    /* 一键报警规则下的报警规则id  */
    PolicyId string `json:"policyId"`

    /*   */
    RuleOption RuleOption `json:"ruleOption"`

    /*  (Optional) */
    WebHookOption WebHookOption `json:"webHookOption"`
}
