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

package edgecloud

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/pkg/utils"
)

type EdgeCloudClientConfig struct {
	cpcfg cloudprovider.ProviderConfig

	authURL      string
	accessKey    string
	accessSecret string
	groupId      string
	billId       string

	token string
	debug bool
}

func NewEdgeCloudClientConfig(authURL, accessKey, accessSecret, groupId, billId string) *EdgeCloudClientConfig {
	cfg := &EdgeCloudClientConfig{
		authURL:      strings.TrimSuffix(authURL, "/"),
		accessKey:    accessKey,
		accessSecret: accessSecret,
		groupId:      groupId,
		billId:       billId,
	}
	return cfg
}

func (cfg *EdgeCloudClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *EdgeCloudClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *EdgeCloudClientConfig) Debug(debug bool) *EdgeCloudClientConfig {
	cfg.debug = debug
	return cfg
}

type SEdgeCloudClient struct {
	*EdgeCloudClientConfig

	httpClient *http.Client

	iregions []cloudprovider.ICloudRegion
}

func NewEdgeCloudClient(cfg *EdgeCloudClientConfig) (*SEdgeCloudClient, error) {
	httpClient := cfg.cpcfg.AdaptiveTimeoutHttpClient()
	cli := &SEdgeCloudClient{
		EdgeCloudClientConfig: cfg,
		httpClient:            httpClient,
	}
	_, _, err := cli.GetRegions(10, 1)
	if err != nil {
		return nil, err
	}
	return cli, nil
}

//func (cli *SEdgeCloudClient) auth() error {
//	authURL := fmt.Sprintf("%s/%s", cli.authURL, "officialcenterccmp/identcenter/auth/authenticatesys")
//	hdr := http.Header{}
//	hdr.Set("platformKey", cli.platformKey)
//	hdr.Set("platformSecret", cli.platformSecret)
//	body := jsonutils.Marshal(map[string]interface{}{
//		"channelCode": cli.channelCode,
//		"account":     cli.account,
//	})
//	_, resp, err := httputils.JSONRequest(cli.httpClient, context.Background(), httputils.POST, authURL, hdr, body, cli.debug)
//	if err != nil {
//		return err
//	}
//	ret := struct {
//		Entity  string
//		Success string
//		Message string
//		Error   string
//	}{}
//	err = resp.Unmarshal(&ret)
//	if err != nil {
//		return errors.Wrapf(err, "resp.Unmarshal")
//	}
//	if len(ret.Entity) == 0 {
//		return fmt.Errorf(resp.String())
//	}
//	cli.token = ret.Entity
//	return nil
//}

func (cli *SEdgeCloudClient) GetRegion(regionId string) *SRegion {
	return &SRegion{cli: cli}
}

func (self *SEdgeCloudClient) _request(method httputils.THttpMethod, uri string, header http.Header, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	timestemp := time.Now().UnixMilli()
	requestId := utils.GenRequestId(16)
	sign, err := func() (string, error) {
		urlInfo, err := url.Parse(uri)
		if err != nil {
			return "", errors.Wrapf(err, "url.Parse(%s)", uri)
		}
		path := urlInfo.Path
		if !strings.HasSuffix(path, "/") {
			path = path + "/"
		}
		path = strings.TrimPrefix(path, "/cmp/v1.0")
		keys := []string{}
		for key := range urlInfo.Query() {
			keys = append(keys, key)
		}
		sort.Sort(sort.StringSlice(keys))
		params := []string{}
		for _, key := range keys {
			params = append(params, fmt.Sprintf("%s=%s", key, urlInfo.Query().Get(key)))
		}
		signStr := fmt.Sprintf("%s%s%d%s%s%s", method, path, timestemp, self.accessKey, requestId, strings.Join(params, "&"))
		log.Errorf("method: %s", method)
		log.Errorf("path: %s", path)
		log.Errorf("params: %s", params)
		log.Errorf("signStr: %s", signStr)
		log.Errorf("accessSecret: %s", self.accessSecret)

		h := hmac.New(sha256.New, []byte(self.accessSecret))
		h.Write([]byte(signStr))
		return base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(h.Sum(nil)))), nil
	}()
	log.Errorf("sign: %s", sign)
	req, _ := http.NewRequest(string(method), uri, nil)
	req.Header.Set("Cmp-Sign", sign)
	//req.Header.Set("Cmp-Sign", url.QueryEscape(sign))
	req.Header.Set("Cmp-AccessKey", self.accessKey)
	req.Header.Set("Cmp-Timestamp", fmt.Sprintf("%d", timestemp))
	req.Header.Set("Cmp-Request-Id", requestId)

	/*
		header.Set("Cmp-Sign", url.QueryEscape(sign))
		header.Set("Cmp-AccessKey", self.accessKey)
		header.Set("Cmp-Timestamp", fmt.Sprintf("%d", timestemp))
		header.Set("Cmp-Request-Id", requestId)
	*/
	_resp, err := self.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	_, resp, err := httputils.ParseJSONResponse("", _resp, nil, true)
	//_, resp, err := httputils.JSONRequest(self.httpClient, context.Background(), method, uri, header, body, self.debug)
	if err != nil {
		if e, ok := err.(*httputils.JSONClientError); ok {
			return nil, fmt.Errorf(e.Details)
		}
		return nil, err
	}
	ret := struct {
		Success string
		Message string
		Error   string
	}{}
	resp.Unmarshal(&ret)
	if ret.Success != "1" || len(ret.Error) > 0 {
		return nil, fmt.Errorf(resp.String())
	}
	return resp, nil
}

func (self *SEdgeCloudClient) request(method httputils.THttpMethod, url string, header http.Header, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	resp, err := self._request(method, url, header, body)
	if err == nil {
		return resp, nil
	}
	//if strings.Contains(err.Error(), "临时令牌鉴权失败") {
	//	e := self.auth()
	//	if e == nil {
	//		return self._request(method, url, header, body)
	//	}
	//}
	return resp, err
}

func (self *SEdgeCloudClient) list(resource string, params url.Values, retVal interface{}) (int, error) {
	url := fmt.Sprintf("%s/cmp/v1.0/%s", self.authURL, resource)
	if len(params) > 0 {
		url = fmt.Sprintf("%s?%s", url, params.Encode())
	}
	hdr := http.Header{}
	resp, err := self.request(httputils.GET, url, hdr, nil)
	if err != nil {
		return 0, err
	}
	total, _ := resp.Int("entity", "total")
	entity, err := resp.Get("entity")
	if err != nil {
		return 0, errors.Wrapf(err, "missing entity")
	}
	if entity.String() == "null" {
		return 0, nil
	}
	_, ok := entity.(*jsonutils.JSONDict)
	if ok && entity.Contains("records") {
		return int(total), resp.Unmarshal(retVal, "entity", "records")
	}
	if ok && entity.Contains("services") {
		return int(total), resp.Unmarshal(retVal, "entity", "services")
	}
	_, ok = entity.(*jsonutils.JSONArray)
	if ok {
		return int(total), resp.Unmarshal(retVal, "entity")
	}
	return 0, fmt.Errorf("invalid resp %s", resp.PrettyString())
}
