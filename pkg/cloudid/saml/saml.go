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

package saml

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudid/options"
	"yunion.io/x/onecloud/pkg/util/samlutils"
	"yunion.io/x/onecloud/pkg/util/samlutils/idp"
)

const (
	SAML_PREFIX = "api/saml/idp"
)

func InitHandlers(app *appsrv.Application) {
	entityId := fmt.Sprintf("https://%s:%d", options.Options.Address, options.Options.Port)
	saml, err := samlutils.NewSAMLInstance(entityId, options.Options.SslCertfile, options.Options.SslKeyfile)
	if err != nil {
		return
	}
	idpInst := idp.NewIdpInstance(saml, nil, nil, nil)
	idpInst.AddHandlers(app, SAML_PREFIX)
}
