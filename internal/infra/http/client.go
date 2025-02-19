/*
 * Copyright 2024 Tech4Works
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package http

import (
	"context"
	"github.com/tech4works/checker"
	"github.com/tech4works/gopen-gateway/internal/app"
	"github.com/tech4works/gopen-gateway/internal/domain/model/vo"
	"go.elastic.co/apm/module/apmhttp/v2"
	"go.elastic.co/apm/v2"
	"io"
	net "net/http"
)

type client struct {
}

func NewClient() app.HTTPClient {
	return client{}
}

func (c client) MakeRequest(ctx context.Context, request *vo.HTTPBackendRequest) (*net.Response, error) {
	httpRequest, err := c.buildNetHTTPRequest(ctx, request)
	if checker.NonNil(err) {
		return nil, err
	}

	netClient := &net.Client{}
	tx := apm.TransactionFromContext(ctx)
	if checker.NonNil(tx) {
		netClient.Transport = apmhttp.WrapRoundTripper(net.DefaultTransport)
	}

	return netClient.Do(httpRequest)
}

func (c client) buildNetHTTPRequest(ctx context.Context, request *vo.HTTPBackendRequest) (*net.Request, error) {
	var body io.ReadCloser
	if request.HasBody() {
		body = io.NopCloser(request.Body().Buffer())
	}
	netReq, err := net.NewRequestWithContext(ctx, request.Method(), request.Url(), body)
	if checker.NonNil(err) {
		return nil, err
	}

	header := request.Header()
	query := request.Query()

	netReq.Header = header.Http()
	netReq.URL.RawQuery = query.Encode()

	return apmhttp.RequestWithContext(ctx, netReq), nil
}
