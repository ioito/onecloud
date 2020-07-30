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

package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SamlProviderCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SamlProviderCreateTask{})
}

func (self *SamlProviderCreateTask) taskFailed(ctx context.Context, sp *models.SSAMLProvider, err error) {
	sp.SetStatus(self.GetUserCred(), api.SAML_PROVIDER_STATUS_CREAT_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, sp, logclient.ACT_CREATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.Marshal(err))
}

func (self *SamlProviderCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	sp := obj.(*models.SSAMLProvider)
	account, err := sp.GetCloudaccount()
	if err != nil {
		self.taskFailed(ctx, sp, errors.Wrap(err, "GetCloudaccoun"))
		return
	}
	factory, err := account.GetProviderFactory()
	if err != nil {
		self.taskFailed(ctx, sp, errors.Wrap(err, "GetProviderFactory"))
		return
	}
	if factory.IsSamlProviderLocalize() {
		sp.SetStatus(self.GetUserCred(), api.SAML_PROVIDER_STATUS_READY, "")
		self.SetStageComplete(ctx, nil)
		return
	}
	provider, err := account.GetProvider()
	if err != nil {
		self.taskFailed(ctx, sp, errors.Wrap(err, "GetProvider"))
		return
	}
	opts := cloudprovider.SAMLProviderCreateOptions{
		Name: sp.Name,
		Desc: sp.Description,
	}
	opts.Metadata, err = sp.GetEntityDescripter()
	if err != nil {
		self.taskFailed(ctx, sp, errors.Wrap(err, "GetEntityDescripter"))
		return
	}
	iSAMLProvider, err := provider.CreateSAMLProvider(&opts)
	if err != nil {
		self.taskFailed(ctx, sp, errors.Wrap(err, "CreateSAMLProvider"))
		return
	}
	db.SetExternalId(sp, self.GetUserCred(), iSAMLProvider.GetGlobalId())
	self.SetStageComplete(ctx, nil)
}
