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

package azure

import (
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SRedisCache struct {
	multicloud.SElasticcacheBase
	multicloud.AzureTags

	region     *SRegion
	ID         string `json:"id"`
	Location   string `json:"location"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Properties struct {
		Provisioningstate string `json:"provisioningState"`
		Redisversion      string `json:"redisVersion"`
		Sku               struct {
			Name     string `json:"name"`
			Family   string `json:"family"`
			Capacity int    `json:"capacity"`
		} `json:"sku"`
		Enablenonsslport bool `json:"enableNonSslPort"`
		Instances        []struct {
			Sslport  int  `json:"sslPort"`
			Shardid  int  `json:"shardId"`
			Ismaster bool `json:"isMaster"`
		} `json:"instances"`
		Publicnetworkaccess string `json:"publicNetworkAccess"`
		Redisconfiguration  struct {
			Maxclients                     string `json:"maxclients"`
			MaxmemoryReserved              string `json:"maxmemory-reserved"`
			MaxfragmentationmemoryReserved string `json:"maxfragmentationmemory-reserved"`
			MaxmemoryDelta                 string `json:"maxmemory-delta"`
		} `json:"redisConfiguration"`
		Accesskeys    interface{}   `json:"accessKeys"`
		Hostname      string        `json:"hostName"`
		Port          int           `json:"port"`
		Sslport       int           `json:"sslPort"`
		Shardcount    int           `json:"shardCount"`
		SubnetId      string        `json:"subnetId"`
		StaticIP      string        `json:"staticIP"`
		Linkedservers []interface{} `json:"linkedServers"`
	} `json:"properties"`
}

func (self *SRegion) GetRedisCache(id string) (*SRedisCache, error) {
	cache := &SRedisCache{region: self}
	return cache, self.get(id, url.Values{}, cache)
}

func (self *SRegion) GetRedisCaches() ([]SRedisCache, error) {
	redis := []SRedisCache{}
	err := self.list("Microsoft.Cache/redis", url.Values{}, &redis)
	if err != nil {
		return nil, errors.Wrapf(err, "list")
	}
	return redis, nil
}

func (self *SRedisCache) GetId() string {
	return self.ID
}

func (self *SRedisCache) GetName() string {
	return self.Name
}

func (self *SRedisCache) GetStatus() string {
	switch self.Properties.Provisioningstate {
	case "Creating":
		return api.ELASTIC_CACHE_STATUS_DEPLOYING
	case "Deleting":
		return api.ELASTIC_CACHE_STATUS_DELETING
	case "Disabled":
		return api.ELASTIC_CACHE_STATUS_INACTIVE
	case "Failed":
		return api.ELASTIC_CACHE_STATUS_CREATE_FAILED
	case "Linking":
		return api.ELASTIC_CACHE_STATUS_RUNNING
	case "Provisioning":
		return api.ELASTIC_CACHE_STATUS_RUNNING
	case "RecoveringScaleFailure":
		return api.ELASTIC_CACHE_STATUS_CHANGE_FAILED
	case "Scaling":
		return api.ELASTIC_CACHE_STATUS_CHANGING
	case "Succeeded":
		return api.ELASTIC_CACHE_STATUS_RUNNING
	case "Unlinking":
		return api.ELASTIC_CACHE_STATUS_RUNNING
	case "Unprovisioning":
		return api.ELASTIC_CACHE_STATUS_RUNNING
	case "Updating":
		return api.ELASTIC_CACHE_STATUS_RUNNING
	default:
		return strings.ToLower(self.Properties.Provisioningstate)
	}
}

func (self *SRedisCache) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SRedisCache) GetInstanceType() string {
	return self.Properties.Sku.Name
}

func (self *SRedisCache) GetCapacityMB() int {
	switch self.Properties.Sku.Family {
	case "P":
		switch self.Properties.Sku.Capacity {
		case 1:
			return 6 * 1024
		case 2:
			return 13 * 1024
		case 3:
			return 26 * 1024
		case 4:
			return 53 * 1024
		case 5:
			return 120 * 1024
		}
	case "C":
		switch self.Properties.Sku.Capacity {
		case 0:
			return 250
		case 1:
			return 1024
		case 2:
			return 2.5 * 1024
		case 3:
			return 6 * 1024
		case 4:
			return 13 * 1024
		case 5:
			return 26 * 1024
		case 6:
			return 53 * 1024
		}
	case "E":
		switch self.Properties.Sku.Capacity {
		case 10:
			return 12 * 1024
		case 20:
			return 25 * 1024
		case 50:
			return 50 * 1024
		case 100:
			return 100 * 1024
		}
	}
	return 0
}

func (self *SRedisCache) GetArchType() string {
	switch self.Properties.Sku.Family {
	case "E", "P":
		return api.ELASTIC_CACHE_ARCH_TYPE_CLUSTER
	}
	return api.ELASTIC_CACHE_ARCH_TYPE_SINGLE
}

func (self *SRedisCache) GetNodeType() string {
	switch len(self.Properties.Instances) {
	case 1:
		return api.ELASTIC_CACHE_NODE_TYPE_SINGLE
	case 2:
		return api.ELASTIC_CACHE_NODE_TYPE_DOUBLE
	case 3:
		return api.ELASTIC_CACHE_NODE_TYPE_THREE
	case 4:
		return api.ELASTIC_CACHE_NODE_TYPE_FOUR
	case 5:
		return api.ELASTIC_CACHE_NODE_TYPE_FIVE
	case 6:
		return api.ELASTIC_CACHE_NODE_TYPE_SIX
	}
	return fmt.Sprintf("%d", self.Properties.Shardcount)
}

func (self *SRedisCache) GetEngine() string {
	return "Redis"
}

func (self *SRedisCache) GetEngineVersion() string {
	return self.Properties.Redisversion
}

func (self *SRedisCache) GetVpcId() string {
	return ""
}

func (self *SRedisCache) GetZoneId() string {
	return ""
}

func (self *SRedisCache) GetNetworkType() string {
	if len(self.Properties.SubnetId) > 0 {
		return api.LB_NETWORK_TYPE_VPC
	}
	return api.LB_NETWORK_TYPE_CLASSIC
}

func (self *SRedisCache) GetNetworkId() string {
	return strings.ToLower(self.Properties.SubnetId)
}

func (self *SRedisCache) GetPrivateDNS() string {
	return ""
}

func (self *SRedisCache) GetPrivateIpAddr() string {
	return self.Properties.StaticIP
}

func (self *SRedisCache) GetPrivateConnectPort() int {
	return self.Properties.Port
}

func (self *SRedisCache) GetPublicDNS() string {
	return self.Properties.Hostname
}

func (self *SRedisCache) GetPublicIpAddr() string {
	return ""
}

func (self *SRedisCache) GetPublicConnectPort() int {
	return self.Properties.Sslport
}

func (self *SRedisCache) GetMaintainStartTime() string {
	return ""
}

func (self *SRedisCache) GetMaintainEndTime() string {
	return ""
}

func (self *SRedisCache) AllocatePublicConnection(port int) (string, error) {
	return "", errors.Wrapf(cloudprovider.ErrNotImplemented, "AllocatePublicConnection")
}

func (self *SRedisCache) ChangeInstanceSpec(spec string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "ChangeInstanceSpec")
}

func (self *SRedisCache) CreateAccount(account cloudprovider.SCloudElasticCacheAccountInput) (cloudprovider.ICloudElasticcacheAccount, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateAccount")
}

func (self *SRedisCache) CreateAcl(aclName, securityIps string) (cloudprovider.ICloudElasticcacheAcl, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateAcl")
}

func (self *SRedisCache) CreateBackup(desc string) (cloudprovider.ICloudElasticcacheBackup, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateBackup")
}

func (self *SRedisCache) Delete() error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "Delete")
}

func (self *SRedisCache) FlushInstance(input cloudprovider.SCloudElasticCacheFlushInstanceInput) error {
	return errors.Wrapf(cloudprovider.ErrNotSupported, "FlushInstance")
}

func (self *SRedisCache) GetAuthMode() string {
	return "on"
}

func (self *SRedisCache) GetICloudElasticcacheAccounts() ([]cloudprovider.ICloudElasticcacheAccount, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudElasticcacheAccounts")
}

func (self *SRedisCache) GetICloudElasticcacheAcls() ([]cloudprovider.ICloudElasticcacheAcl, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudElasticcacheAcls")
}

func (self *SRedisCache) GetICloudElasticcacheBackups() ([]cloudprovider.ICloudElasticcacheBackup, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudElasticcacheBackups")
}

func (self *SRedisCache) GetICloudElasticcacheParameters() ([]cloudprovider.ICloudElasticcacheParameter, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudElasticcacheParameters")
}

func (self *SRedisCache) GetICloudElasticcacheAccount(accountId string) (cloudprovider.ICloudElasticcacheAccount, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudElasticcacheAccount")
}

func (self *SRedisCache) GetICloudElasticcacheAcl(aclId string) (cloudprovider.ICloudElasticcacheAcl, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudElasticcacheAcl")
}

func (self *SRedisCache) GetICloudElasticcacheBackup(backupId string) (cloudprovider.ICloudElasticcacheBackup, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudElasticcacheBackup")
}

func (self *SRedisCache) GetSecurityGroupIds() ([]string, error) {
	return []string{}, nil
}

func (self *SRedisCache) ReleasePublicConnection() error {
	return cloudprovider.ErrNotSupported
}

func (self *SRedisCache) Restart() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SRedisCache) SetMaintainTime(start, end string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SRedisCache) UpdateAuthMode(noPasswordAccess bool, password string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SRedisCache) UpdateBackupPolicy(config cloudprovider.SCloudElasticCacheBackupPolicyUpdateInput) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SRedisCache) UpdateInstanceParameters(config jsonutils.JSONObject) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SRedisCache) UpdateSecurityGroups(secgroupIds []string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetIElasticcaches() ([]cloudprovider.ICloudElasticcache, error) {
	redis, err := self.GetRedisCaches()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRedisCaches")
	}
	ret := []cloudprovider.ICloudElasticcache{}
	for i := range redis {
		redis[i].region = self
		ret = append(ret, &redis[i])
	}
	return ret, nil
}

func (self *SRegion) GetIElasticcacheById(id string) (cloudprovider.ICloudElasticcache, error) {
	return self.GetRedisCache(id)
}
