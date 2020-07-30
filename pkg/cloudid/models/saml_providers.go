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

package models

import (
	"context"
	"encoding/base64"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/options"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/samlutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"
)

type SSAMLProviderManager struct {
	db.SExternalizedResourceBaseManager
	db.SStatusInfrasResourceBaseManager
	SCloudaccountResourceBaseManager
}

var SamlProviderManager *SSAMLProviderManager

func init() {
	SamlProviderManager = &SSAMLProviderManager{
		SStatusInfrasResourceBaseManager: db.NewStatusInfrasResourceBaseManager(
			SSAMLProvider{},
			"saml_providers_tbl",
			"saml_provider",
			"saml_providers",
		),
	}
	SamlProviderManager.SetVirtualObject(SamlProviderManager)
}

type SSAMLProvider struct {
	db.SStatusInfrasResourceBase
	db.SExternalizedResourceBase
	SCloudaccountResourceBase

	EntityId         string `charset:"ascii" list:"domain" create:"domain_optional"`
	MetadataDocument string `charset:"ascii" list:"domain" create:"domain_required"`
}

func (manager *SSAMLProviderManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (manager *SSAMLProviderManager) GetIVirtualModelManager() db.IVirtualModelManager {
	return manager.GetVirtualObject().(db.IVirtualModelManager)
}

func (manager *SSAMLProviderManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.SAMLProviderListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (manager *SSAMLProviderManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowCreate(userCred, manager)
}

// 创建身份提供商
func (manager *SSAMLProviderManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.SAMLProviderCreateInput) (api.SAMLProviderCreateInput, error) {
	if len(input.CloudaccountId) == 0 {
		return input, httperrors.NewMissingParameterError("cloudaccount_id")
	}
	account, err := CloudaccountManager.FetchAccount(ctx, input.CloudaccountId)
	if err != nil {
		return input, err
	}
	input.CloudaccountId = account.Id

	if len(input.MetadataDocument) == 0 {
		return input, httperrors.NewMissingParameterError("metadata_document")
	}

	metadata, err := base64.StdEncoding.DecodeString(input.MetadataDocument)
	if err != nil {
		return input, httperrors.NewInputParameterError("invalid base64 metadata document")
	}
	entityDescriptor, err := samlutils.ParseMetadata([]byte(metadata))
	if err != nil {
		return input, httperrors.NewInputParameterError("invalid entity descriptor")
	}

	input.EntityId = string(entityDescriptor.EntityId)
	if input.EntityId != options.Options.BaseOptions.GetEntityId() {
		return input, httperrors.NewInputParameterError("")
	}

	q := manager.Query().Equals("entity_id", input.EntityId).Equals("cloudaccount_id", input.CloudaccountId).Equals("status", api.SAML_PROVIDER_STATUS_READY)
	sps := []SSAMLProvider{}
	err = db.FetchModelObjects(manager, q, &sps)
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrap(err, "db.FetchModelObjects"))
	}

	if len(sps) > 0 {
		return input, httperrors.NewDuplicateResourceError("account %s has been registed saml provider %s", account.Name, sps[0].Name)
	}

	input.StatusInfrasResourceBaseCreateInput, err = manager.SStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (self *SSAMLProvider) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.StartSamlProviderCreateTask(ctx, userCred, "")
}

func (self *SSAMLProvider) StartSamlProviderCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "SamlProviderCreateTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.SAML_PROVIDER_STATUS_CREATING, "")
	task.ScheduleRun(nil)
	return nil
}

// 获取身份提供商详情
func (self *SSAMLProvider) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.SAMLProviderDetails, error) {
	return api.SAMLProviderDetails{}, nil
}

func (manager *SSAMLProviderManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.SAMLProviderDetails {
	rows := make([]api.SAMLProviderDetails, len(objs))
	sinf := manager.SStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.SAMLProviderDetails{
			StatusInfrasResourceBaseDetails: sinf[i],
		}
	}
	return rows
}

func (manager *SSAMLProviderManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

// 删除身份提供商
func (self *SSAMLProvider) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartSamlProviderDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (self *SSAMLProvider) StartSamlProviderDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "SamlProviderDeleteTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(userCred, api.SAML_PROVIDER_STATUS_DELETING, "")
	task.ScheduleRun(nil)
	return nil
}

func (self *SSAMLProvider) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SSAMLProvider) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SSAMLProvider) GetICloudSAMLProvider() (cloudprovider.ICloudSAMLProvider, error) {
	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudaccount")
	}
	provider, err := account.GetProvider()
	if err != nil {
		return nil, errors.Wrap(err, "GetProvider")
	}
	return provider.GetICloudSAMLProviderById(self.ExternalId)
}

func (self *SSAMLProvider) GetEntityDescripter() (samlutils.EntityDescriptor, error) {
	ed := samlutils.EntityDescriptor{}
	metadata, err := base64.StdEncoding.DecodeString(self.MetadataDocument)
	if err != nil {
		return ed, errors.Wrap(err, "invalid base64 metadata document")
	}
	return samlutils.ParseMetadata([]byte(metadata))
}
