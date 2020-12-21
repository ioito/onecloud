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

package google

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

type SRegionSkuPrice struct {
	Name     string
	Desc     string
	IsCustom bool
	Extended bool
	Price    float64
}

type SRegionPrice struct {
	StoragePrice []SRegionSkuPrice
	CpuPrice     []SRegionSkuPrice
	RamPrice     []SRegionSkuPrice
	SkuPrice     []SRegionSkuPrice
}

type SkuPrice struct {
	Region    string
	PriceInfo SRegionPrice
}

func (self *SRegion) ListSkuPrice() (*SkuPrice, error) {
	skus := []SSkuBilling{}
	params := map[string]string{}
	err := self.BillingList("services/6F81-5844-456A/skus", params, 0, "", &skus)
	if err != nil {
		return nil, err
	}
	result := SkuPrice{
		Region: self.Name,
	}
	for _, sku := range skus {
		if !utils.IsInStringArray(self.Name, sku.ServiceRegions) {
			continue
		}
		if sku.ServiceProviderName != "Google" {
			continue
		}
		if sku.Category.ResourceFamily != "Compute" && sku.Category.ResourceFamily != "Storage" {
			continue
		}
		if sku.Category.ResourceFamily == "Storage" && strings.Contains(sku.Description, "Regional") {
			continue
		}
		if sku.Category.ResourceGroup == "CPU" && strings.Contains(sku.Description, "Sole") {
			continue
		}
		if sku.Category.UsageType != "OnDemand" {
			continue
		}
		price := SRegionSkuPrice{
			Desc: sku.Description,
		}
		for _, p := range sku.PricingInfo {
			for _, rate := range p.PricingExpression.TieredRates {
				if rate.UnitPrice.Nanos == 0 {
					continue
				}
				price.Price = float64(rate.UnitPrice.Nanos) / 1000000000
				break
			}
		}
		if strings.Contains(sku.Description, "Custom") {
			price.IsCustom = true
		}
		if strings.Contains(sku.Description, "Extended") {
			price.Extended = true
		}
		switch sku.Category.ResourceGroup {
		case "N1Standard":
			price.Name = "n1"
			if strings.Contains(sku.Description, "Core") {
				result.PriceInfo.CpuPrice = append(result.PriceInfo.CpuPrice, price)
			} else if strings.Contains(sku.Description, "Ram") {
				result.PriceInfo.RamPrice = append(result.PriceInfo.RamPrice, price)
			} else {
				return nil, fmt.Errorf("invalid n1 price sku info %s", jsonutils.Marshal(sku).PrettyString())
			}
		case "F1Micro":
			price.Name = "f1-micro"
			result.PriceInfo.SkuPrice = append(result.PriceInfo.SkuPrice, price)
		case "G1Small":
			price.Name = "g1-small"
			result.PriceInfo.SkuPrice = append(result.PriceInfo.SkuPrice, price)
		case "CPU", "RAM":
			for prefix, v := range map[string]string{
				"Memory Optimized Upgrade Premium": "m2",
				"E2":                               "e2",
				"N2D AMD":                          "n2d",
				"N2":                               "n2",
				"Memory-optimized":                 "m1",
				"Compute optimized":                "c2",
				"N1 Predefined":                    "n1",
				"Custom Instance":                  "n1-custom",
				"Custom Extended Instance":         "n1-custom",
			} {
				if strings.HasPrefix(sku.Description, prefix) {
					price.Name = v
					break
				}
			}
			if sku.Category.ResourceGroup == "CPU" {
				result.PriceInfo.CpuPrice = append(result.PriceInfo.CpuPrice, price)
			} else {
				result.PriceInfo.RamPrice = append(result.PriceInfo.RamPrice, price)
			}
		case "PDStandard":
			price.Name = api.STORAGE_GOOGLE_PD_STANDARD
			result.PriceInfo.StoragePrice = append(result.PriceInfo.StoragePrice, price)
		case "SSD":
			price.Name = api.STORAGE_GOOGLE_PD_SSD
			if strings.Contains(sku.Description, "Balanced PD") {
				price.Name = api.STORAGE_GOOGLE_PD_BALANCED
			}
			result.PriceInfo.StoragePrice = append(result.PriceInfo.StoragePrice, price)
		case "LocalSSD":
			price.Name = api.STORAGE_GOOGLE_LOCAL_SSD
			result.PriceInfo.StoragePrice = append(result.PriceInfo.StoragePrice, price)
		case "PDSnapshot", "StorageImage", "GPU", "MachineImage":
			continue
		default:
			return nil, fmt.Errorf("new resource group %s", sku.Category.ResourceGroup)
		}
	}
	return &result, nil
}

type SkuInfo struct {
	Type        string
	Discount    []float64
	Cpu         int
	CpuRate     float64
	MemMb       int
	MemRate     float64
	ExtendRate  float64
	ExtendMemMb int
}

func (self *SkuPrice) Estimate(sku string) (float64, float64, float64, error) {
	var hour, month, year float64
	skuInfo := strings.Split(sku, "-")
	if len(skuInfo) < 2 || len(skuInfo) > 4 {
		return hour, month, year, fmt.Errorf("invalid sku %s", sku)
	}
	info := SkuInfo{Discount: []float64{0.0, 0.2, 0.4, 0.6}}
	info.Type = skuInfo[0]
	if utils.IsInStringArray("custom", skuInfo) {
		info.MemMb, _ = strconv.Atoi(skuInfo[len(skuInfo)-1])
		info.Cpu, _ = strconv.Atoi(skuInfo[len(skuInfo)-2])
		switch info.Type {
		case "custom", "n1":
			info.Type = "n1-custom"
			info.ExtendMemMb = info.MemMb - info.Cpu*int(6.5*1024)
		case "n2":
			info.ExtendMemMb = info.MemMb - info.Cpu*8*1024
			info.Discount = []float64{0.0, 0.132, 0.267, 0.4}
		case "n2d":
			info.ExtendMemMb = info.MemMb - info.Cpu*8*1024
			info.Discount = []float64{0.0, 0.132, 0.267, 0.4}
		default:
			return hour, month, year, fmt.Errorf("invalid custom sku %s", sku)
		}
		if info.ExtendMemMb < 0 {
			info.ExtendMemMb = 0
		} else {
			info.MemMb = info.MemMb - info.ExtendMemMb
		}
		for _, rate := range self.PriceInfo.CpuPrice {
			if rate.IsCustom && rate.Name == info.Type {
				info.CpuRate = rate.Price
				break
			}
		}
		for _, rate := range self.PriceInfo.RamPrice {
			if rate.IsCustom && rate.Name == info.Type {
				if rate.Extended {
					info.ExtendRate = rate.Price
				} else {
					info.MemRate = rate.Price
				}
			}
		}
		log.Errorf("cpu %d %f mem: %d %f extend: %d %f discount: %v", info.Cpu, info.CpuRate, info.MemMb/1024, info.MemRate, info.ExtendMemMb, info.ExtendRate, info.Discount)
		hour = info.CpuRate*float64(info.Cpu) + float64(info.MemMb)*info.MemRate/1024 + float64(info.ExtendMemMb)*info.ExtendRate/1024
		month = 182.5 * hour
		month += 182.5 * (1 - info.Discount[1]) * hour
		month += 182.5 * (1 - info.Discount[2]) * hour
		month += 172.49999999999997 * (1 - info.Discount[3]) * hour
		year = month * 12
		return hour, month, year, nil
	} else if utils.IsInStringArray(sku, []string{"e2-micro", "e2-small", "e2-medium", "f1-micro", "g1-small"}) {
		if !utils.IsInStringArray(sku, []string{"f1-micro", "g1-small"}) {
			url := "https://cloudpricingcalculator.appspot.com/static/data/pricelist.json"
			_, resp, err := httputils.JSONRequest(httputils.GetDefaultClient(), context.Background(), "GET", url, nil, nil, false)
			if err != nil {
				return hour, month, year, errors.Wrapf(err, "httputils.JSONRequest")
			}
			key := fmt.Sprintf("CP-COMPUTEENGINE-VMIMAGE-%s", strings.ToUpper(sku))
			price, err := resp.Float("gcp_price_list", key, self.Region)
			if err != nil {
				return hour, month, year, errors.Wrapf(err, "resp.Float")
			}
			hour := price
			month = hour * 720
			year = month * 12
			return hour, month, year, nil
		}
		for _, rate := range self.PriceInfo.SkuPrice {
			if rate.Name == sku {
				hour = rate.Price
				month = 182.5 * hour
				month += 182.5 * (1 - info.Discount[1]) * hour
				month += 182.5 * (1 - info.Discount[2]) * hour
				month += 172.49999999999997 * (1 - info.Discount[3]) * hour
				year = month * 12
				return hour, month, year, nil
			}
		}
		return hour, month, year, fmt.Errorf("not found sku price")
	} else {
		info.Cpu, _ = strconv.Atoi(skuInfo[len(skuInfo)-1])
		memInfo := map[string]float32{}
		switch info.Type {
		case "n1":
			memInfo = map[string]float32{"standard": 3.75, "highmem": 6.5, "highcpu": 0.9}
		case "n2":
			memInfo = map[string]float32{"standard": 4, "highmem": 8, "highcpu": 1}
			info.Discount = []float64{0.0, 0.132, 0.267, 0.4}
		case "e2":
			memInfo = map[string]float32{"standard": 4, "highmem": 8, "highcpu": 1}
			info.Discount = []float64{0.0, 0.0, 0.0, 0.0}
		case "n2d":
			memInfo = map[string]float32{"highmem": 8, "highcpu": 1}
			info.Discount = []float64{0.0, 0.132, 0.267, 0.4}
		case "m1":
			memInfo = map[string]float32{"ultramem": 24.025, "megamem": 14.93}
		case "m2":
			memInfo = map[string]float32{"ultramem": 28.3}
		case "c2":
			memInfo = map[string]float32{"standard": 4}
			info.Discount = []float64{0.0, 0.132, 0.267, 0.4}
		default:
			return hour, month, year, fmt.Errorf("invalid sku %s", sku)
		}
		if memPerCpu, ok := memInfo[skuInfo[1]]; ok {
			info.MemMb = info.Cpu * int(memPerCpu*1024)
		} else {
			return 0.0, 0.0, 0.0, fmt.Errorf("unknow n1 sku %s", sku)
		}
		for _, rate := range self.PriceInfo.CpuPrice {
			if !rate.IsCustom && rate.Name == info.Type {
				info.CpuRate = rate.Price
				break
			}
		}
		for _, rate := range self.PriceInfo.RamPrice {
			if !rate.IsCustom && rate.Name == info.Type && !rate.Extended {
				info.MemRate = rate.Price
				break
			}
		}
		log.Errorf("cpu %d %f mem: %d %f", info.Cpu, info.CpuRate, info.MemMb/1024, info.MemRate)
		hour = info.CpuRate*float64(info.Cpu) + float64(info.MemMb)*info.MemRate/1024
		month = 182.5 * hour
		month += 182.5 * (1 - info.Discount[1]) * hour
		month += 182.5 * (1 - info.Discount[2]) * hour
		month += 172.49999999999997 * (1 - info.Discount[3]) * hour
		year = month * 12
	}
	return hour, month, year, nil
}
