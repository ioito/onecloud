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
	"strconv"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DomianListOptions struct {
		Offset int
		Limit  int
	}
	shellutils.R(&DomianListOptions{}, "domain-list", "List domains", func(cli *qcloud.SRegion, args *DomianListOptions) error {
		domains, total, e := cli.GetClient().GetDomains(args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(domains, total, args.Offset, args.Limit, []string{})
		// cli.GetClient().GetAllDomains()
		return nil
	})

	type DomianCreateOptions struct {
		Domain string
	}
	shellutils.R(&DomianCreateOptions{}, "domain-create", "create domain", func(cli *qcloud.SRegion, args *DomianCreateOptions) error {
		domain, e := cli.GetClient().CreateDomian(args.Domain)
		if e != nil {
			return e
		}
		printObject(domain)
		return nil
	})

	type DomianDeleteOptions struct {
		Domain string
	}
	shellutils.R(&DomianDeleteOptions{}, "domain-delete", "delete domains", func(cli *qcloud.SRegion, args *DomianDeleteOptions) error {
		e := cli.GetClient().DeleteDomian(args.Domain)
		if e != nil {
			return e
		}
		return nil
	})

	type DnsRecordListOptions struct {
		Domain string
		Offset int
		Limit  int
	}
	shellutils.R(&DnsRecordListOptions{}, "dnsrecord-list", "List dndrecord", func(cli *qcloud.SRegion, args *DnsRecordListOptions) error {
		records, total, e := cli.GetClient().GetDnsRecords(args.Domain, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(records, total, args.Offset, args.Limit, []string{})
		// cli.GetClient().GetAllDnsRecords(args.Domain)
		return nil
	})

	type DnsRecordCreateOptions struct {
		Domain string
		Name   string
		Value  string //joined by '*'
		Ttl    int64
		Type   string
	}
	shellutils.R(&DnsRecordCreateOptions{}, "dnsrecord-create", "create dndrecord", func(cli *qcloud.SRegion, args *DnsRecordCreateOptions) error {
		change := cloudprovider.DnsRecordSet{}
		change.DnsName = args.Name
		change.DnsValue = args.Value
		change.Ttl = args.Ttl
		change.DnsType = cloudprovider.TDnsType(args.Type)
		e := cli.GetClient().CreateDnsRecord(&change, args.Domain)
		if e != nil {
			return e
		}
		return nil
	})

	type DnsRecordUpdateOptions struct {
		Domain   string
		Name     string
		RecordId int
		Value    string //joined by '*'
		Ttl      int64
		Type     string
	}
	shellutils.R(&DnsRecordUpdateOptions{}, "dnsrecord-update", "update dndrecord", func(cli *qcloud.SRegion, args *DnsRecordUpdateOptions) error {
		change := cloudprovider.DnsRecordSet{}
		change.DnsName = args.Name
		change.ExternalId = strconv.Itoa(args.RecordId)
		change.DnsValue = args.Value
		change.Ttl = args.Ttl
		change.DnsType = cloudprovider.TDnsType(args.Type)
		e := cli.GetClient().ModifyDnsRecord(&change, args.Domain)
		if e != nil {
			return e
		}
		return nil
	})

	type DnsRecordRemoveOptions struct {
		Domain   string
		RecordId int
	}
	shellutils.R(&DnsRecordRemoveOptions{}, "dnsrecord-delete", "delete dndrecord", func(cli *qcloud.SRegion, args *DnsRecordRemoveOptions) error {
		e := cli.GetClient().DeleteDnsRecord(args.RecordId, args.Domain)
		if e != nil {
			return e
		}
		return nil
	})
}
