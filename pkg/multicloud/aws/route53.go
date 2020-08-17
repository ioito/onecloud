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

package aws

import (
	"regexp"

	"github.com/aws/aws-sdk-go/aws/request"
)

var restHandler = request.NamedHandler{Name: "awssdk.restxml.Build", Fn: restBuild}

var reSanitizeURL = regexp.MustCompile(`\/%2F\w+%2F`)

func restBuild(r *request.Request) {
	//r.HTTPRequest.URL.RawPath =
	//	reSanitizeURL.ReplaceAllString(r.HTTPRequest.URL.RawPath, "/")

	//updated, err := url.Parse(r.HTTPRequest.URL.RawPath)
	//if err != nil {
	//	r.Error = awserr.New(request.ErrCodeSerialization, "failed to clean Route53 URL", err)
	//	return
	//}

	//r.HTTPRequest.URL.Path = updated.Path
	//rest.Build(r)

	//if t := rest.PayloadType(r.Params); t == "structure" || t == "" {
	//	var buf bytes.Buffer
	//	err := xmlutil.BuildXML(r.Params, xml.NewEncoder(&buf))
	//	if err != nil {
	//		r.Error = awserr.NewRequestFailure(
	//			awserr.New(request.ErrCodeSerialization,
	//				"failed to encode rest XML request", err),
	//			0,
	//			r.RequestID,
	//		)
	//		return
	//	}
	//	r.SetBufferBody(buf.Bytes())
	//}
}
