/*
 * Copyright 2024 Gabriel Cataldo
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

package api

import (
	"bytes"
	"fmt"
	"github.com/GabrielHCataldo/go-errors/errors"
	"github.com/GabrielHCataldo/go-helper/helper"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/tech4works/gopen-gateway/internal/app"
	"github.com/tech4works/gopen-gateway/internal/app/model/dto"
	"github.com/tech4works/gopen-gateway/internal/domain/mapper"
	"github.com/tech4works/gopen-gateway/internal/domain/model/vo"
	"github.com/uber/jaeger-client-go"
	"golang.org/x/net/context"
	"io"
	"sync"
	"time"
)

type Context struct {
	startTime time.Time
	span      opentracing.Span
	mutex     *sync.RWMutex
	engine    *gin.Context
	gopen     *vo.Gopen
	endpoint  *vo.Endpoint
	request   *vo.HTTPRequest
	response  *vo.HTTPResponse
}

func newContext(gin *gin.Context, gopen *vo.Gopen, endpoint *vo.Endpoint) app.Context {
	request := buildHTTPRequest(gin)
	return &Context{
		startTime: time.Now(),
		span:      buildSpan(gin, request),
		mutex:     &sync.RWMutex{},
		engine:    gin,
		gopen:     gopen,
		endpoint:  endpoint,
		request:   request,
	}
}

func buildSpan(gin *gin.Context, request *vo.HTTPRequest) opentracing.Span {
	var span opentracing.Span

	tracer := opentracing.GlobalTracer()
	wireContext, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(gin.Request.Header))
	if helper.IsNotNil(err) {
		span = opentracing.StartSpan(gin.FullPath())
	} else {
		span = opentracing.StartSpan(gin.FullPath(), ext.RPCServerOption(wireContext))
	}
	gin.Request = gin.Request.WithContext(opentracing.ContextWithSpan(gin.Request.Context(), span))

	span.SetTag("request.url", request.Url())
	span.SetTag("request.method", request.Method())
	span.SetTag("request.params", request.Params().String())
	span.SetTag("request.header", request.Header().String())
	span.SetTag("request.query", request.Query().String())
	if helper.IsNotNil(request.Body()) {
		s, _ := request.Body().String()
		span.SetTag("request.body", helper.SimpleCompactString(s))
	} else {
		span.SetTag("request.body", "")
	}

	return span
}

func buildHTTPRequest(gin *gin.Context) *vo.HTTPRequest {
	gin.Request.Header.Add(mapper.XForwardedFor, gin.ClientIP())
	header := vo.NewHeader(gin.Request.Header)

	query := vo.NewQuery(gin.Request.URL.Query())
	url := gin.Request.URL.Path
	if helper.IsNotEmpty(query) {
		url = fmt.Sprint(url, "?", query.Encode())
	}

	ginParams := map[string]string{}
	for _, param := range gin.Params {
		ginParams[param.Key] = param.Value
	}
	path := vo.NewURLPath(gin.FullPath(), ginParams)

	bodyBytes, _ := io.ReadAll(gin.Request.Body)
	body := vo.NewBody(gin.GetHeader(mapper.ContentType), gin.GetHeader(mapper.ContentEncoding), bytes.NewBuffer(bodyBytes))

	return vo.NewHTTPRequest(path, url, gin.Request.Method, header, query, body)
}

func (c *Context) Context() context.Context {
	return c.engine.Request.Context()
}

func (c *Context) Done() <-chan struct{} {
	return c.Context().Done()
}

func (c *Context) WithContext(ctx context.Context) {
	c.engine.Request = c.engine.Request.WithContext(ctx)
}

func (c *Context) Next() {
	c.engine.Next()
}

func (c *Context) Latency() time.Duration {
	return time.Now().Sub(c.startTime)
}

func (c *Context) Span() opentracing.Span {
	return c.span
}

func (c *Context) TraceID() string {
	spanContext, ok := c.span.Context().(jaeger.SpanContext)
	if ok {
		return spanContext.TraceID().String()
	}
	return "undefined"
}

func (c *Context) ClientIP() string {
	return c.Request().Header().GetFirst(mapper.XForwardedFor)
}

func (c *Context) Gopen() *vo.Gopen {
	return c.gopen
}

func (c *Context) Endpoint() *vo.Endpoint {
	return c.endpoint
}

func (c *Context) Request() *vo.HTTPRequest {
	return c.request
}

func (c *Context) Response() *vo.HTTPResponse {
	return c.response
}

func (c *Context) Write(response *vo.HTTPResponse) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.engine.IsAborted() {
		return
	}

	var contentType vo.ContentType
	var rawBodyBytes []byte
	if response.HasBody() {
		contentType = response.Body().ContentType()
		rawBodyBytes = response.Body().RawBytes()
	}

	c.writeHeader(response.Header())
	if helper.IsNotEmpty(rawBodyBytes) {
		c.writeBody(response.StatusCode(), contentType.String(), rawBodyBytes)
	} else {
		c.writeStatusCode(response.StatusCode())
	}

	c.transformToWritten(response)
}

func (c *Context) WriteError(code int, err error) {
	statusCode := vo.NewStatusCode(code)

	details := errors.Details(err)
	buffer := helper.SimpleConvertToBuffer(dto.ErrorBody{
		File:      details.GetFile(),
		Line:      details.GetLine(),
		Endpoint:  c.endpoint.Path(),
		Message:   details.GetMessage(),
		Timestamp: time.Now(),
	})
	body := vo.NewBodyJson(buffer)
	header := c.buildHeader(false, statusCode, body)

	c.Write(vo.NewHTTPResponse(statusCode, header, body))
}

func (c *Context) WriteCacheResponse(cacheResponse *vo.CacheResponse) {
	c.Write(vo.NewHTTPResponse(cacheResponse.StatusCode, c.buildCacheHeader(cacheResponse), cacheResponse.Body))
}

func (c *Context) WriteStatusCode(code int) {
	statusCode := vo.NewStatusCode(code)
	header := c.buildHeader(true, statusCode, nil)

	c.Write(vo.NewHTTPResponseStatusCode(statusCode, header))
}

func (c *Context) WriteString(code int, s string) {
	statusCode := vo.NewStatusCode(code)
	body := vo.NewBodyWithContentType(vo.NewContentTypeTextPlain(), helper.SimpleConvertToBuffer(s))
	header := c.buildHeader(true, statusCode, body)

	c.Write(vo.NewHTTPResponse(statusCode, header, body))
}

func (c *Context) WriteJson(code int, a any) {
	statusCode := vo.NewStatusCode(code)
	body := vo.NewBodyWithContentType(vo.NewContentTypeJson(), helper.SimpleConvertToBuffer(a))
	header := c.buildHeader(true, statusCode, body)

	c.Write(vo.NewHTTPResponse(statusCode, header, body))
}

func (c *Context) buildHeader(complete bool, statusCode vo.StatusCode, body *vo.Body) vo.Header {
	mapHeader := map[string][]string{
		mapper.XGopenCache:    {"false"},
		mapper.XGopenComplete: {helper.SimpleConvertToString(complete)},
		mapper.XGopenSuccess:  {helper.SimpleConvertToString(statusCode.OK())},
	}
	if helper.IsNotNil(body) {
		mapHeader[mapper.ContentType] = []string{body.ContentType().String()}
		mapHeader[mapper.ContentLength] = []string{body.LenStr()}
	}
	return vo.NewHeader(mapHeader)
}

func (c *Context) buildCacheHeader(cacheResponse *vo.CacheResponse) vo.Header {
	copied := cacheResponse.Header.Copy()
	copied[mapper.XGopenCache] = []string{"true"}
	copied[mapper.XGopenCacheTTL] = []string{cacheResponse.TTL()}
	return vo.NewHeader(copied)
}

func (c *Context) writeStatusCode(statusCode vo.StatusCode) {
	if c.engine.IsAborted() {
		return
	}
	c.engine.Status(statusCode.Code())
}

func (c *Context) writeHeader(header vo.Header) {
	for _, key := range header.Keys() {
		c.engine.Header(key, header.Get(key))
	}
}

func (c *Context) writeBody(statusCode vo.StatusCode, contentType string, body []byte) {
	if c.engine.IsAborted() {
		return
	}
	c.engine.Data(statusCode.Code(), contentType, body)
}

func (c *Context) transformToWritten(response *vo.HTTPResponse) {
	c.engine.Abort()
	c.response = response

	statusCode := response.StatusCode()
	header := response.Header()
	body := response.Body()

	span := c.Span()
	span.SetTag("response.status", statusCode.String())
	span.SetTag("response.header", header.String())
	if helper.IsNotNil(body) {
		s, _ := body.String()
		span.SetTag("response.body", helper.SimpleCompactString(s))
	} else {
		span.SetTag("response.body", "")
	}
	span.Finish()
}
