package factory

import (
	"github.com/GabrielHCataldo/go-helper/helper"
	"github.com/tech4works/checker"
	"github.com/tech4works/gopen-gateway/internal/domain/mapper"
	"github.com/tech4works/gopen-gateway/internal/domain/model/enum"
	"github.com/tech4works/gopen-gateway/internal/domain/model/vo"
	"github.com/tech4works/gopen-gateway/internal/domain/service"
	"net/http"
)

type httpResponseFactory struct {
	aggregatorService   service.Aggregator
	omitterService      service.Omitter
	nomenclatureService service.Nomenclature
	contentService      service.Content
	httpBackendFactory  HTTPBackend
}

type HTTPResponse interface {
	BuildAbortedResponse(endpoint *vo.Endpoint, history *vo.History) *vo.HTTPResponse
	BuildResponse(endpoint *vo.Endpoint, history *vo.History) (*vo.HTTPResponse, []error)
}

func NewHTTPResponse(aggregatorService service.Aggregator, omitterService service.Omitter,
	nomenclatureService service.Nomenclature, contentService service.Content, httpBackendFactory HTTPBackend) HTTPResponse {
	return httpResponseFactory{
		aggregatorService:   aggregatorService,
		omitterService:      omitterService,
		nomenclatureService: nomenclatureService,
		contentService:      contentService,
		httpBackendFactory:  httpBackendFactory,
	}
}

func (h httpResponseFactory) BuildAbortedResponse(endpoint *vo.Endpoint, history *vo.History) *vo.HTTPResponse {
	lastBackendResponse := history.Last()
	lastStatusCode := lastBackendResponse.StatusCode()
	lastHeader := lastBackendResponse.Header()
	lastBody := lastBackendResponse.Body()

	header := vo.NewHeader(map[string][]string{
		mapper.XGopenCache:    {"false"},
		mapper.XGopenSuccess:  {helper.SimpleConvertToString(lastStatusCode.OK())},
		mapper.XGopenComplete: {helper.SimpleConvertToString(checker.Equals(history.Size(), endpoint.CountBackendsNonOmit()))},
	})
	header = h.aggregatorService.AggregateHeaders(header, lastHeader)

	return vo.NewHTTPResponse(lastStatusCode, header, lastBody)
}

func (h httpResponseFactory) BuildResponse(endpoint *vo.Endpoint, history *vo.History) (*vo.HTTPResponse, []error) {
	var allErrs []error

	statusCode := h.buildStatusCodeByHistory(history)
	body, bodyErrs := h.buildBodyByHistory(endpoint, history)
	header := h.buildHeaderByHistory(endpoint, body, history)

	allErrs = append(allErrs, bodyErrs...)

	return vo.NewHTTPResponse(statusCode, header, body), allErrs
}

func (h httpResponseFactory) buildStatusCodeByHistory(history *vo.History) vo.StatusCode {
	if history.MultipleResponses() {
		return h.buildStatusCodeFromMultipleResponses(history)
	} else if history.SingleResponse() {
		return history.Last().StatusCode()
	}
	return vo.NewStatusCode(http.StatusNoContent)
}

func (h httpResponseFactory) buildBodyByHistory(endpoint *vo.Endpoint, history *vo.History) (*vo.Body, []error) {
	var body *vo.Body
	var errs []error

	if history.MultipleResponses() {
		body, errs = h.buildBodyFromMultipleResponses(endpoint, history)
	} else {
		body = history.Last().Body()
	}

	if !endpoint.HasResponse() {
		return body, nil
	}

	body, omitErrs := h.omitEmptyValuesFromBody(endpoint.Response().OmitEmpty(), body)
	body, modifyCaseErrs := h.modifyBodyCase(endpoint.Response().Nomenclature(), body)
	body, modifyContentTypeErrs := h.modifyBodyContentType(endpoint.Response().ContentType(), body)
	body, modifyBodyContentEncodingErrs := h.modifyBodyContentEncoding(endpoint.Response().ContentEncoding(), body)

	errs = append(errs, omitErrs...)
	errs = append(errs, modifyCaseErrs...)
	errs = append(errs, modifyContentTypeErrs...)
	errs = append(errs, modifyBodyContentEncodingErrs...)

	return body, errs
}

func (h httpResponseFactory) buildHeaderByHistory(endpoint *vo.Endpoint, body *vo.Body, history *vo.History) vo.Header {
	mapHeader := map[string][]string{
		mapper.XGopenCache:    {"false"},
		mapper.XGopenSuccess:  {helper.SimpleConvertToString(history.AllOK())},
		mapper.XGopenComplete: {helper.SimpleConvertToString(checker.Equals(history.Size(), endpoint.CountBackendsNonOmit()))},
	}
	if checker.NonNil(body) {
		mapHeader[mapper.ContentType] = []string{body.ContentType().String()}
		mapHeader[mapper.ContentLength] = []string{body.SizeInString()}
		if body.HasContentEncoding() {
			mapHeader[mapper.ContentEncoding] = []string{body.ContentEncoding().String()}
		}
	}

	header := vo.NewHeader(mapHeader)

	for i := 0; i < history.Size(); i++ {
		_, _, httpBackendResponse := history.Get(i)
		header = h.aggregatorService.AggregateHeaders(header, httpBackendResponse.Header())
	}

	return header
}

func (h httpResponseFactory) buildBodyFromMultipleResponses(endpoint *vo.Endpoint, history *vo.History) (*vo.Body, []error) {
	if endpoint.HasResponse() && endpoint.Response().Aggregate() {
		return h.aggregatorService.AggregateBodies(history)
	}
	return h.aggregatorService.AggregateBodiesIntoSlice(history)
}

func (h httpResponseFactory) buildStatusCodeFromMultipleResponses(history *vo.History) vo.StatusCode {
	statusCodes := make(map[vo.StatusCode]int)
	for i := 0; i < history.Size(); i++ {
		_, _, httpBackendResponse := history.Get(i)
		statusCodes[httpBackendResponse.StatusCode()]++
	}

	mostFrequentCode := vo.NewStatusCode(http.StatusNoContent)
	maxCount := 0
	for statusCode, count := range statusCodes {
		if count >= maxCount {
			mostFrequentCode = statusCode
			maxCount = count
		}
	}

	return mostFrequentCode
}

func (h httpResponseFactory) omitEmptyValuesFromBody(omitEmpty bool, body *vo.Body) (*vo.Body, []error) {
	if !omitEmpty {
		return body, nil
	}
	return h.omitterService.OmitEmptyValuesFromBody(body)
}

func (h httpResponseFactory) modifyBodyCase(nomenclature enum.Nomenclature, body *vo.Body) (*vo.Body, []error) {
	if !nomenclature.IsEnumValid() {
		return body, nil
	}
	return h.nomenclatureService.ToCase(body, nomenclature)
}

func (h httpResponseFactory) modifyBodyContentType(contentTypeConfig enum.ContentType, body *vo.Body) (*vo.Body, []error) {
	var contentType enum.ContentType
	if contentTypeConfig.IsEnumValid() {
		contentType = contentTypeConfig
	} else {
		contentType = body.ContentType().ToEnum()
	}

	newBody, err := h.contentService.ModifyBodyContentType(body, contentType)
	if checker.NonNil(err) {
		return body, []error{err}
	}

	return newBody, nil
}

func (h httpResponseFactory) modifyBodyContentEncoding(contentEncodingConfig enum.ContentEncoding, body *vo.Body) (
	*vo.Body, []error) {
	var contentEncoding enum.ContentEncoding
	if contentEncodingConfig.IsEnumValid() {
		contentEncoding = contentEncodingConfig
	} else if body.ContentEncoding().IsGzip() {
		contentEncoding = enum.ContentEncodingGzip
	} else if body.ContentEncoding().IsDeflate() {
		contentEncoding = enum.ContentEncodingDeflate
	} else {
		contentEncoding = enum.ContentEncodingNone
	}

	newBody, err := h.contentService.ModifyBodyContentEncoding(body, contentEncoding)
	if checker.NonNil(err) {
		return body, []error{err}
	}

	return newBody, nil
}
