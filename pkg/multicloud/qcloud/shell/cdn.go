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

package shell

import (
	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type CdnDomainCreateOption struct {
		Domain           string
		OriginType       string
		Origins          []string
		CosPrivateAccess string
	}

	shellutils.R(&CdnDomainCreateOption{}, "cdn-domain-create", "create cdn domain", func(cli *qcloud.SRegion, args *CdnDomainCreateOption) error {
		err := cli.GetClient().AddCdnDomain(args.Domain, args.OriginType, args.Origins, args.CosPrivateAccess)
		if err != nil {
			return err
		}
		return nil
	})
}