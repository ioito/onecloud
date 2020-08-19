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
	"yunion.io/x/onecloud/pkg/multicloud/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type HostedZoneListOptions struct{}
	shellutils.R(&HostedZoneListOptions{}, "hostedzone-list", "List hostedzone", func(cli *aws.SRegion, args *HostedZoneListOptions) error {
		hostzones, err := cli.GetClient().GetHostedZones()
		if err != nil {
			return err
		}
		printList(hostzones, len(hostzones), 0, 20, []string{})
		return nil
	})
	type DnsRecordSetListOptions struct {
		HostedzoneId string
	}
	shellutils.R(&DnsRecordSetListOptions{}, "dnsrecordset-list", "List dnsrecordset", func(cli *aws.SRegion, args *DnsRecordSetListOptions) error {
		dnsrecordsets, err := cli.GetClient().GetSdnsRecordSets(args.HostedzoneId)
		if err != nil {
			return err
		}
		printList(dnsrecordsets, len(dnsrecordsets), 0, 20, []string{})
		return nil
	})
	type TrafficPolicyGetOptions struct {
		TrafficPolicyId string
	}
	shellutils.R(&TrafficPolicyGetOptions{}, "trafficpolicy-list", "List trafficpolicy", func(cli *aws.SRegion, args *TrafficPolicyGetOptions) error {
		trafficpolicy, err := cli.GetClient().GetSTrafficPolicyById(args.TrafficPolicyId)
		if err != nil {
			return err
		}
		printList(trafficpolicy, 1, 0, 20, []string{})
		return nil
	})
}
