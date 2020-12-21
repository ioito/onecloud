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
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/multicloud/google"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SkuBillingListOptions struct {
		PageSize  int
		PageToken string
	}
	shellutils.R(&SkuBillingListOptions{}, "sku-billing-list", "List sku billing", func(cli *google.SRegion, args *SkuBillingListOptions) error {
		billings, err := cli.ListSkuBilling(args.PageSize, args.PageToken)
		if err != nil {
			return err
		}
		printList(billings, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&SkuBillingListOptions{}, "compute-sku-billing-list", "List sku billing", func(cli *google.SRegion, args *SkuBillingListOptions) error {
		billings, err := cli.ListSkuBilling(args.PageSize, args.PageToken)
		if err != nil {
			return err
		}
		info := cli.GetSkuRateInfo(billings)
		fmt.Println(jsonutils.Marshal(info).PrettyString())
		return nil
	})

	type SkuEstimate struct {
		SKU string
	}

	shellutils.R(&SkuEstimate{}, "sku-estimate", "Estimate sku price", func(cli *google.SRegion, args *SkuEstimate) error {
		result, err := cli.ListSkuPrice()
		if err != nil {
			return err
		}
		hour, month, year, err := result.Estimate(args.SKU)
		if err != nil {
			return err
		}
		fmt.Printf("hour %f, month: %f year: %f", hour, month, year)
		return nil
	})

	type RegionPriceInfo struct {
	}

	shellutils.R(&RegionPriceInfo{}, "sku-price-list", "Show region sku info", func(cli *google.SRegion, args *RegionPriceInfo) error {
		result, err := cli.ListSkuPrice()
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

}
