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

package server

import (
	"context"
	"fmt"
	"github.com/tech4works/checker"
	"github.com/tech4works/converter"
	"github.com/tech4works/gopen-gateway/internal/app"
	"github.com/tech4works/gopen-gateway/internal/app/controller"
	"github.com/tech4works/gopen-gateway/internal/app/factory"
	"github.com/tech4works/gopen-gateway/internal/app/middleware"
	"github.com/tech4works/gopen-gateway/internal/app/model/dto"
	"github.com/tech4works/gopen-gateway/internal/app/usecase"
	"github.com/tech4works/gopen-gateway/internal/domain"
	domainFactory "github.com/tech4works/gopen-gateway/internal/domain/factory"
	"github.com/tech4works/gopen-gateway/internal/domain/model/vo"
	"github.com/tech4works/gopen-gateway/internal/domain/service"
	"go.elastic.co/apm/module/apmhttp/v2"
	net "net/http"
	"os"
)

type http struct {
	net                     *net.Server
	gopen                   *vo.Gopen
	log                     app.BootLog
	router                  app.Router
	panicRecoveryMiddleware middleware.PanicRecovery
	logMiddleware           middleware.Log
	securityCorsMiddleware  middleware.SecurityCors
	timeoutMiddleware       middleware.Timeout
	limiterMiddleware       middleware.Limiter
	cacheMiddleware         middleware.Cache
	staticController        controller.Static
	endpointController      controller.Endpoint
}

type HTTP interface {
	ListenAndServe()
	Shutdown(ctx context.Context) error
}

func New(
	gopen *dto.Gopen,
	log app.BootLog,
	router app.Router,
	httpClient app.HTTPClient,
	endpointLog app.EndpointLog,
	backendLog app.BackendLog,
	httpLog app.HTTPLog,
	jsonPath domain.JSONPath,
	converter domain.Converter,
	store domain.Store,
	nomenclature domain.Nomenclature,
) HTTP {
	log.PrintInfo("Building domain...")
	mapperService := service.NewMapper(jsonPath)
	projectorService := service.NewProjector(jsonPath)
	dynamicValueService := service.NewDynamicValue(jsonPath)
	modifierService := service.NewModifier(jsonPath)
	omitterService := service.NewOmitter(jsonPath)
	nomenclatureService := service.NewNomenclature(jsonPath, nomenclature)
	contentService := service.NewContent(converter)
	aggregatorService := service.NewAggregator(jsonPath)
	limiterService := service.NewLimiter()
	securityCorsService := service.NewSecurityCors()
	cacheService := service.NewCache(store)

	log.PrintInfo("Building factories...")
	httpBackendFactory := domainFactory.NewHTTPBackend(mapperService, projectorService, dynamicValueService,
		modifierService, omitterService, nomenclatureService, contentService, aggregatorService)
	httpResponseFactory := domainFactory.NewHTTPResponse(aggregatorService, omitterService, nomenclatureService,
		contentService, httpBackendFactory)

	log.PrintInfo("Building use cases...")
	endpointUseCase := usecase.NewEndpoint(httpBackendFactory, httpResponseFactory, httpClient, endpointLog, backendLog)

	log.PrintInfo("Building middlewares...")
	panicRecoveryMiddleware := middleware.NewPanicRecovery(endpointLog)
	logMiddleware := middleware.NewLog(httpLog)
	securityCorsMiddleware := middleware.NewSecurityCors(securityCorsService)
	timeoutMiddleware := middleware.NewTimeout()
	limiterMiddleware := middleware.NewLimiter(limiterService)
	cacheMiddleware := middleware.NewCache(cacheService, endpointLog)

	log.PrintInfo("Building controllers...")
	staticController := controller.NewStatic(gopen)
	endpointController := controller.NewEndpoint(endpointUseCase)

	log.PrintInfo("Building value objects...")
	return &http{
		gopen:                   factory.BuildGopen(gopen),
		log:                     log,
		router:                  router,
		panicRecoveryMiddleware: panicRecoveryMiddleware,
		logMiddleware:           logMiddleware,
		timeoutMiddleware:       timeoutMiddleware,
		limiterMiddleware:       limiterMiddleware,
		cacheMiddleware:         cacheMiddleware,
		securityCorsMiddleware:  securityCorsMiddleware,
		staticController:        staticController,
		endpointController:      endpointController,
	}
}

func (h *http) ListenAndServe() {
	h.log.PrintInfo("Configuring routes...")

	h.buildStaticRoutes()
	h.buildRoutes()

	h.net = &net.Server{
		Addr:    fmt.Sprint(":", os.Getenv("GOPEN_PORT")),
		Handler: apmhttp.Wrap(h.router.Engine()),
	}

	h.log.SkipLine()
	h.log.PrintTitle(fmt.Sprintf("LISTEN AND SERVE %s", h.net.Addr))

	h.net.ListenAndServe()
}

func (h *http) Shutdown(ctx context.Context) error {
	if checker.IsNil(h.net) {
		return nil
	}
	return h.net.Shutdown(ctx)
}

func (h *http) buildRoutes() {
	for _, endpoint := range h.gopen.Endpoints() {
		handles := h.buildEndpointHandles()
		h.router.Handle(h.gopen, &endpoint, handles...)

		h.log.PrintInfof("Registered route with %s handles: %s", converter.ToString(len(handles)), endpoint.Resume())
	}
}

func (h *http) buildStaticRoutes() {
	formatLog := "Registered route with 5 handles: %s --> \"%s\""

	pingEndpoint := h.buildStaticPingRoute()
	h.log.PrintInfof(formatLog, pingEndpoint.Method(), pingEndpoint.Path())

	versionEndpoint := h.buildStaticVersionRoute()
	h.log.PrintInfof(formatLog, versionEndpoint.Method(), versionEndpoint.Path())

	settingsEndpoint := h.buildStaticSettingsRoute()
	h.log.PrintInfof(formatLog, settingsEndpoint.Method(), settingsEndpoint.Path())
}

func (h *http) buildStaticPingRoute() *vo.Endpoint {
	endpoint := vo.NewEndpointStatic("/ping", net.MethodGet)
	h.buildStaticRoute(&endpoint, h.staticController.Ping)
	return &endpoint
}

func (h *http) buildStaticVersionRoute() *vo.Endpoint {
	endpoint := vo.NewEndpointStatic("/version", net.MethodGet)
	h.buildStaticRoute(&endpoint, h.staticController.Version)
	return &endpoint
}

func (h *http) buildStaticSettingsRoute() *vo.Endpoint {
	endpoint := vo.NewEndpointStatic("/settings", net.MethodGet)
	h.buildStaticRoute(&endpoint, h.staticController.Settings)
	return &endpoint
}

func (h *http) buildStaticRoute(endpointStatic *vo.Endpoint, handler app.HandlerFunc) {
	timeoutHandler := h.timeoutMiddleware.Do
	panicHandler := h.panicRecoveryMiddleware.Do
	logHandler := h.logMiddleware.Do
	limiterHandler := h.limiterMiddleware.Do
	h.router.Handle(h.gopen, endpointStatic, timeoutHandler, panicHandler, logHandler, limiterHandler, handler)
}

func (h *http) buildEndpointHandles() []app.HandlerFunc {
	return []app.HandlerFunc{
		h.timeoutMiddleware.Do,
		h.panicRecoveryMiddleware.Do,
		h.logMiddleware.Do,
		h.securityCorsMiddleware.Do,
		h.limiterMiddleware.Do,
		h.cacheMiddleware.Do,
		h.endpointController.Execute,
	}
}
