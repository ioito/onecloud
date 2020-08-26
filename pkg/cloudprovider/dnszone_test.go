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

package cloudprovider

import "testing"

func TestDiff(t *testing.T) {
	cases := []struct {
		Name        string
		Remote      DnsRecordSets
		Local       DnsRecordSets
		CommonCount int
		AddCount    int
		DelCount    int
		UpdateCount int
	}{
		{
			Name:        "Test delete",
			CommonCount: 4,
			AddCount:    0,
			DelCount:    1,
			UpdateCount: 0,
			Remote: []DnsRecordSet{
				DnsRecordSet{
					ExternalId: "650124294",
					Enabled:    true,
					DnsName:    "@",
					DnsType:    DnsTypeNS,
					DnsValue:   "f1g1ns1.dnspod.net.",
					Ttl:        86400,
					PolicyType: DnsPolicyTypeSimple,
				},
				DnsRecordSet{
					ExternalId: "650124301",
					Enabled:    true,
					DnsName:    "@",
					DnsType:    DnsTypeNS,
					DnsValue:   "f1g1ns2.dnspod.net.",
					Ttl:        86400,
					PolicyType: DnsPolicyTypeSimple,
				},
				DnsRecordSet{
					ExternalId: "650124650",
					Enabled:    true,
					DnsName:    "@",
					DnsType:    DnsTypeMX,
					DnsValue:   "qiye163mx01.mxmail.netease.com.",
					Ttl:        600,
					PolicyType: DnsPolicyTypeSimple,
				},
				DnsRecordSet{
					ExternalId: "650124659",
					Enabled:    true,
					DnsName:    "@",
					DnsType:    DnsTypeMX,
					DnsValue:   "qiye163mx02.mxmail.netease.com.",
					Ttl:        600,
					PolicyType: DnsPolicyTypeSimple,
				},
				DnsRecordSet{
					ExternalId: "650124661",
					Enabled:    true,
					DnsName:    "mail",
					DnsType:    DnsTypeCNAME,
					DnsValue:   "qiye.163.com.",
					Ttl:        600,
					PolicyType: DnsPolicyTypeSimple,
				},
			},

			Local: []DnsRecordSet{
				DnsRecordSet{
					Id:         "d599c0e0-0653-40ed-85e1-86502a8d23d4",
					Enabled:    true,
					DnsName:    "mail",
					DnsType:    DnsTypeCNAME,
					DnsValue:   "qiye.163.com.",
					Ttl:        600,
					PolicyType: DnsPolicyTypeSimple,
				},
				DnsRecordSet{
					Id:         "5728b06e-f8cb-41eb-86e9-0e5836195ad1",
					Enabled:    true,
					DnsName:    "@",
					DnsType:    DnsTypeMX,
					DnsValue:   "qiye163mx01.mxmail.netease.com.",
					Ttl:        600,
					PolicyType: DnsPolicyTypeSimple,
				},
				DnsRecordSet{
					Id:         "427b38d2-77e2-4705-8880-0852da9cfb6b",
					Enabled:    true,
					DnsName:    "@",
					DnsType:    DnsTypeNS,
					DnsValue:   "f1g1ns1.dnspod.net.",
					Ttl:        86400,
					PolicyType: DnsPolicyTypeSimple,
				},
				DnsRecordSet{
					Id:         "0390724d-cb49-43f8-8ccd-117fef3f5034",
					Enabled:    true,
					DnsName:    "@",
					DnsType:    DnsTypeNS,
					DnsValue:   "f1g1ns2.dnspod.net.",
					Ttl:        86400,
					PolicyType: DnsPolicyTypeSimple,
				},
			},
		},
	}
	for _, c := range cases {
		iRecords := []ICloudDnsRecordSet{}
		for i := range c.Remote {
			iRecords = append(iRecords, &c.Remote[i])
		}
		common, add, del, update := CompareDnsRecordSet(iRecords, c.Local, true)
		if len(common) != c.CommonCount {
			t.Fatalf("[%s] common should be %d current is %d", c.Name, c.CommonCount, len(common))
		}
		if len(add) != c.AddCount {
			t.Fatalf("[%s] add should be %d current is %d", c.Name, c.AddCount, len(add))
		}
		if len(del) != c.DelCount {
			t.Fatalf("[%s] del should be %d current is %d", c.Name, c.DelCount, len(del))
		}
		if len(update) != c.UpdateCount {
			t.Fatalf("[%s] update should be %d current is %d", c.Name, c.UpdateCount, len(update))
		}
	}
}
