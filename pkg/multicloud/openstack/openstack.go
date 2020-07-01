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

package openstack

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/version"
)

const (
	CLOUD_PROVIDER_OPENSTACK = api.CLOUD_PROVIDER_OPENSTACK
	OPENSTACK_DEFAULT_REGION = "RegionOne"
)

type OpenstackClientConfig struct {
	cpcfg cloudprovider.ProviderConfig

	authURL       string
	username      string
	password      string
	project       string
	projectDomain string

	domainName   string
	endpointType string

	debug bool
}

func NewOpenstackClientConfig(authURL, username, password, project, projectDomain string) *OpenstackClientConfig {
	cfg := &OpenstackClientConfig{
		authURL:       authURL,
		username:      username,
		password:      password,
		project:       project,
		projectDomain: projectDomain,
	}
	return cfg
}

func (cfg *OpenstackClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *OpenstackClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *OpenstackClientConfig) DomainName(domainName string) *OpenstackClientConfig {
	cfg.domainName = domainName
	return cfg
}

func (cfg *OpenstackClientConfig) EndpointType(endpointType string) *OpenstackClientConfig {
	cfg.endpointType = endpointType
	return cfg
}

func (cfg *OpenstackClientConfig) Debug(debug bool) *OpenstackClientConfig {
	cfg.debug = debug
	return cfg
}

type SOpenStackClient struct {
	*OpenstackClientConfig

	client          *mcclient.Client
	tokenCredential mcclient.TokenCredential
	iregions        []cloudprovider.ICloudRegion
}

func NewOpenStackClient(cfg *OpenstackClientConfig) (*SOpenStackClient, error) {
	cli := &SOpenStackClient{
		OpenstackClientConfig: cfg,
	}
	return cli, cli.fetchRegions()
}

func (cli *SOpenStackClient) GetCloudRegionExternalIdPrefix() string {
	return fmt.Sprintf("%s/%s/", CLOUD_PROVIDER_OPENSTACK, cli.cpcfg.CloudaccountId)
}

func (cli *SOpenStackClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	projects, err := cli.GetIProjects()
	if err != nil {
		return nil, errors.Wrap(err, "GetIProjects")
	}
	accounts := []cloudprovider.SSubAccount{}
	for i := range projects {
		subAccount := cloudprovider.SSubAccount{
			Account: fmt.Sprintf("%s/%s", projects[i].GetName(), cli.username),
			Name:    projects[i].GetName(),
		}
		if len(cli.domainName) > 0 {
			subAccount.Account = fmt.Sprintf("%s/%s", subAccount.Account, cli.domainName)
		}
		accounts = append(accounts, subAccount)
	}
	return accounts, nil
}

func (cli *SOpenStackClient) fetchRegions() error {
	if err := cli.connect(); err != nil {
		return err
	}

	regions := cli.tokenCredential.GetRegions()
	cli.iregions = make([]cloudprovider.ICloudRegion, len(regions))
	for i := 0; i < len(regions); i++ {
		region := SRegion{client: cli, Name: regions[i]}
		cli.iregions[i] = &region
	}

	for _, region := range regions {
		if serviceURL, err := cli.tokenCredential.GetServiceURL("compute", region, "", cli.endpointType); err != nil || len(serviceURL) == 0 {
			for _, endpointType := range []string{"internal", "admin", "public"} {
				if serviceURL, err := cli.tokenCredential.GetServiceURL("compute", region, "", endpointType); err == nil && len(serviceURL) > 0 {
					cli.endpointType = endpointType
					return nil
				}
			}
		} else {
			return nil
		}
	}
	return fmt.Errorf("failed to find right endpoint type")
}

func (cli *SOpenStackClient) Request(region, service, method string, url string, microversion string, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	header := http.Header{}
	if len(microversion) > 0 {
		header.Set("X-Openstack-Nova-API-Version", microversion)
	}

	session := cli.client.NewSession(context.Background(), region, "", cli.endpointType, cli.tokenCredential, "")
	serviceUrl, err := session.GetServiceURL(service, "")
	if err != nil {
		return nil, nil, errors.Wrapf(err, "GetServiceURL(%s)", service)
	}
	url = strings.TrimPrefix(url, serviceUrl)
	header, resp, err := session.JSONRequest(service, "", httputils.THttpMethod(method), url, header, body)
	if err != nil && body != nil {
		log.Errorf("microversion %s url: %s, params: %s", microversion, serviceUrl+url, body.PrettyString())
	}
	return header, resp, err
}

func (cli *SOpenStackClient) RawRequest(region, service, method string, url string, microversion string, body jsonutils.JSONObject) (*http.Response, error) {
	header := http.Header{}
	if len(microversion) > 0 {
		header.Set("X-Openstack-Nova-API-Version", microversion)
	}
	ctx := context.Background()
	session := cli.client.NewSession(ctx, region, "", cli.endpointType, cli.tokenCredential, "")
	data := strings.NewReader("")
	if body != nil {
		data = strings.NewReader(body.String())
	}
	return session.RawRequest(service, "", httputils.THttpMethod(method), url, header, data)
}

func (cli *SOpenStackClient) StreamRequest(region, service, method string, url string, microversion string, body io.Reader) (*http.Response, error) {
	header := http.Header{}
	if len(microversion) > 0 {
		header.Set("X-Openstack-Nova-API-Version", microversion)
	}
	header.Set("Content-Type", "application/octet-stream")
	ctx := context.Background()
	session := cli.client.NewSession(ctx, region, "", cli.endpointType, cli.tokenCredential, "")
	return session.RawRequest(service, "", httputils.THttpMethod(method), url, header, body)
}

func (cli *SOpenStackClient) getVersion(region string, service string) (string, string, error) {
	ctx := context.Background()
	session := cli.client.NewSession(ctx, region, "", cli.endpointType, cli.tokenCredential, "")
	uri, err := session.GetServiceURL(service, cli.endpointType)
	if err != nil {
		return "", "", err
	}
	url := uri
	telnetID := cli.tokenCredential.GetTenantId()
	if strings.Index(uri, telnetID) > 0 {
		url = uri[0:strings.Index(uri, telnetID)]
	}
	_, resp, err := session.JSONRequest(url, "", "GET", "/", nil, nil)
	if err != nil {
		return "", "", err
	}
	minVersion, _ := resp.GetString("version", "min_version")
	maxVersion, _ := resp.GetString("version", "version")
	if resp.Contains("versions") {
		minVersion, maxVersion = "1000.0", ""
		versions, _ := resp.GetArray("versions")
		for _, _version := range versions {
			if _minVersion, _ := _version.GetString("min_version"); len(_minVersion) > 0 {
				if version.LT(_minVersion, minVersion) {
					minVersion = _minVersion
				}
			}
			if _maxVersion, _ := _version.GetString("version"); len(_maxVersion) > 0 {
				if version.GT(_maxVersion, maxVersion) {
					maxVersion = _maxVersion
				}
			}
		}
		if minVersion == "1000.0" {
			minVersion, maxVersion = "", ""
		}
	}
	return minVersion, maxVersion, nil
}

func (cli *SOpenStackClient) connect() error {
	cli.client = mcclient.NewClient(cli.authURL, 5, cli.debug, false, "", "")
	cli.client.SetHttpTransportProxyFunc(cli.cpcfg.ProxyFunc)
	tokenCredential, err := cli.client.Authenticate(cli.username, cli.password, cli.domainName, cli.project, cli.projectDomain)
	if err != nil {
		return err
	}
	cli.tokenCredential = tokenCredential
	return nil
}

func (cli *SOpenStackClient) getProjectTokenCredential(projectId string) (mcclient.TokenCredential, error) {
	project, err := cli.GetProject(projectId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetProject(%s)", projectId)
	}
	region := cli.iregions[0].(*SRegion)
	s := cli.client.NewSession(context.Background(), region.Name, "", cli.endpointType, cli.tokenCredential, "")

	roleId, err := modules.RolesV3.GetId(s, "admin", jsonutils.Marshal(map[string]string{}))
	if err != nil {
		return nil, errors.Wrap(err, "RolesV3.GetId")
	}
	_, err = modules.RolesV3.PutInContexts(s, roleId, nil, []modulebase.ManagerContext{{InstanceManager: &modules.Projects, InstanceId: project.GetId()}, {InstanceManager: &modules.UsersV3, InstanceId: s.GetUserId()}})
	if err != nil {
		return nil, errors.Wrap(err, "RolesV3.PutInContexts")
	}

	cli.client = mcclient.NewClient(cli.authURL, 5, cli.debug, false, "", "")
	cli.client.SetHttpTransportProxyFunc(cli.cpcfg.ProxyFunc)
	tokenCredential, err := cli.client.Authenticate(cli.username, cli.password, cli.domainName, project.Name, cli.projectDomain)
	if err != nil {
		return nil, errors.Wrap(err, "Authenticate")
	}
	return tokenCredential, nil
}

func (cli *SOpenStackClient) GetRegion(regionId string) *SRegion {
	for i := 0; i < len(cli.iregions); i++ {
		if cli.iregions[i].GetId() == regionId {
			return cli.iregions[i].(*SRegion)
		}
	}
	return nil
}

func (cli *SOpenStackClient) GetIRegions() []cloudprovider.ICloudRegion {
	return cli.iregions
}

func (cli *SOpenStackClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(cli.iregions); i++ {
		if cli.iregions[i].GetGlobalId() == id {
			return cli.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (cli *SOpenStackClient) GetRegions() []SRegion {
	regions := make([]SRegion, len(cli.iregions))
	for i := 0; i < len(regions); i++ {
		region := cli.iregions[i].(*SRegion)
		regions[i] = *region
	}
	return regions
}

func (cli *SOpenStackClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	if len(cli.iregions) > 0 {
		region := cli.iregions[0].(*SRegion)
		s := cli.client.NewSession(context.Background(), region.Name, "", cli.endpointType, cli.tokenCredential, "")
		result, err := modules.Projects.List(s, jsonutils.NewDict())
		if err != nil {
			return nil, err
		}
		iprojects := []cloudprovider.ICloudProject{}
		for i := 0; i < len(result.Data); i++ {
			project := &SProject{}
			if err := result.Data[i].Unmarshal(project); err != nil {
				return nil, err
			}
			iprojects = append(iprojects, project)
		}
		return iprojects, nil
	}
	return nil, cloudprovider.ErrNotImplemented
}

func (cli *SOpenStackClient) GetProject(id string) (*SProject, error) {
	if len(cli.iregions) > 0 {
		region := cli.iregions[0].(*SRegion)
		s := cli.client.NewSession(context.Background(), region.Name, "", cli.endpointType, cli.tokenCredential, "")
		result, err := modules.Projects.Get(s, id, jsonutils.NewDict())
		if err != nil {
			return nil, err
		}
		project := &SProject{}
		err = result.Unmarshal(project)
		if err != nil {
			return nil, errors.Wrap(err, "result.Unmarshal")
		}
		return project, nil
	}
	return nil, fmt.Errorf("no region info")
}

func (cli *SOpenStackClient) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SOpenStackClient) CreateProject(name, desc string) (*SProject, error) {
	region := cli.iregions[0].(*SRegion)
	s := cli.client.NewSession(context.Background(), region.Name, "", cli.endpointType, cli.tokenCredential, "")

	params := map[string]string{
		"name":      name,
		"domain_id": s.GetDomainId(),
	}
	if len(desc) > 0 {
		params["description"] = desc
	}
	result, err := modules.Projects.Create(s, jsonutils.Marshal(params))
	if err != nil {
		return nil, errors.Wrap(err, "Projects.Create")
	}
	project := SProject{}
	err = result.Unmarshal(&project)
	if err != nil {
		return nil, errors.Wrap(err, "result.Unmarshal")
	}
	return &project, nil
}

func (self *SOpenStackClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_PROJECT,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
		// cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		// cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		// cloudprovider.CLOUD_CAPABILITY_RDS,
		// cloudprovider.CLOUD_CAPABILITY_CACHE,
		// cloudprovider.CLOUD_CAPABILITY_EVENT,
	}
	return caps
}
