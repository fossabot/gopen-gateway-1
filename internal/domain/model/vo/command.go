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

package vo

// ExecuteEndpoint represents the execution context for an API endpoint in the Gopen application.
//
// Fields:
// - gopen: a pointer to the Gopen struct, representing the Gopen server configuration.
// - endpoint: a pointer to the Endpoint struct, representing the specific API endpoint configuration.
// - httpRequest: a pointer to the HttpRequest struct, representing the incoming HTTP request.
type ExecuteEndpoint struct {
	gopen       *Gopen
	endpoint    *Endpoint
	httpRequest *HttpRequest
}

// ExecuteBackend represents the execution context for a backend in the Gopen application.
//
// Fields:
// - endpoint: a pointer to the Endpoint struct, representing the specific API endpoint configuration.
// - backend: a pointer to the Backend struct, representing the backend configuration for an application or service.
// - httpRequest: a pointer to the HttpRequest struct, representing the incoming HTTP request.
// - httpResponse: a pointer to the HttpResponse struct, representing the outgoing HTTP response.
type ExecuteBackend struct {
	endpoint     *Endpoint
	backend      *Backend
	httpRequest  *HttpRequest
	httpResponse *HttpResponse
}

// NewExecuteEndpoint creates a new ExecuteEndpoint using the provided Gopen, Endpoint, and HttpRequest objects.
func NewExecuteEndpoint(gopen *Gopen, endpoint *Endpoint, httpRequest *HttpRequest) *ExecuteEndpoint {
	return &ExecuteEndpoint{
		gopen:       gopen,
		endpoint:    endpoint,
		httpRequest: httpRequest,
	}
}

// NewExecuteBackend creates a new ExecuteBackend using the provided Endpoint, Backend, HttpRequest, and HttpResponse objects.
func NewExecuteBackend(endpoint *Endpoint, backend *Backend, httpRequest *HttpRequest, httpResponse *HttpResponse,
) *ExecuteBackend {
	return &ExecuteBackend{
		endpoint:     endpoint,
		backend:      backend,
		httpRequest:  httpRequest,
		httpResponse: httpResponse,
	}
}

// Endpoint returns the Endpoint object associated with the ExecuteEndpoint object.
func (e ExecuteEndpoint) Endpoint() *Endpoint {
	return e.endpoint
}

// HttpRequest returns the HttpRequest object associated with the ExecuteEndpoint object.
func (e ExecuteEndpoint) HttpRequest() *HttpRequest {
	return e.httpRequest
}

// Gopen returns the Gopen object associated with the ExecuteEndpoint object.
func (e ExecuteEndpoint) Gopen() *Gopen {
	return e.gopen
}

// Endpoint returns the Endpoint object associated with the ExecuteEndpoint object.
func (e ExecuteBackend) Endpoint() *Endpoint {
	return e.endpoint
}

// Backend returns the Backend object associated with the ExecuteBackend object.
func (e ExecuteBackend) Backend() *Backend {
	return e.backend
}

// HttpRequest returns the HttpRequest object associated with the ExecuteBackend object.
func (e ExecuteBackend) HttpRequest() *HttpRequest {
	return e.httpRequest
}

// HttpResponse returns the HttpResponse object associated with the ExecuteBackend object.
func (e ExecuteBackend) HttpResponse() *HttpResponse {
	return e.httpResponse
}
