package service

import (
	"github.com/GabrielHCataldo/go-errors/errors"
	"github.com/tech4works/checker"
	"github.com/tech4works/converter"
	"github.com/tech4works/gopen-gateway/internal/domain"
	"github.com/tech4works/gopen-gateway/internal/domain/mapper"
	"github.com/tech4works/gopen-gateway/internal/domain/model/vo"
	"regexp"
	"strings"
)

type dynamicValueService struct {
	jsonPath domain.JSONPath
}

type DynamicValue interface {
	Get(value string, request *vo.HTTPRequest, history *vo.History) (string, []error)
	GetAsSliceOfString(value string, request *vo.HTTPRequest, history *vo.History) ([]string, []error)
}

func NewDynamicValue(jsonPath domain.JSONPath) DynamicValue {
	return dynamicValueService{
		jsonPath: jsonPath,
	}
}

func (d dynamicValueService) Get(value string, request *vo.HTTPRequest, history *vo.History) (string, []error) {
	var errs []error
	for _, word := range d.findAllBySyntax(value) {
		result, err := d.getValueBySyntax(word, request, history)
		if errors.Is(err, mapper.ErrValueNotFound) {
			continue
		} else if checker.NonNil(err) {
			errs = append(errs, err)
		} else {
			value = strings.Replace(value, word, result, 1)
		}
	}
	return value, errs
}

func (d dynamicValueService) GetAsSliceOfString(value string, request *vo.HTTPRequest, history *vo.History) ([]string, []error) {
	newValue, errs := d.Get(value, request, history)
	if checker.IsSlice(newValue) {
		var ss []string
		err := converter.ToDestWithErr(newValue, &ss)
		if checker.IsNil(err) {
			return ss, errs
		} else {
			errs = append(errs, err)
		}
	}
	return []string{newValue}, errs
}

func (d dynamicValueService) findAllBySyntax(value string) []string {
	regex := regexp.MustCompile(`\B#[a-zA-Z0-9_.\-\[\]]+`)
	return regex.FindAllString(value, -1)
}

func (d dynamicValueService) getValueBySyntax(word string, request *vo.HTTPRequest, history *vo.History) (string, error) {
	cleanSintaxe := strings.ReplaceAll(word, "#", "")
	dotSplit := strings.Split(cleanSintaxe, ".")
	if checker.IsEmpty(dotSplit) {
		return "", errors.Newf("Invalid dynamic value syntax! key: %s", word)
	}

	prefix := dotSplit[0]
	if checker.Contains(prefix, "request") {
		return d.getRequestValueByJsonPath(cleanSintaxe, request)
	} else if checker.Contains(prefix, "responses") {
		return d.getResponseValueByJsonPath(cleanSintaxe, history)
	} else {
		return "", errors.Newf("Invalid prefix syntax %s!", prefix)
	}
}

func (d dynamicValueService) getRequestValueByJsonPath(jsonPath string, request *vo.HTTPRequest) (string, error) {
	jsonPath = strings.Replace(jsonPath, "request.", "", 1)

	jsonRequest, err := request.Map()
	if checker.NonNil(err) {
		return "", err
	}

	result := d.jsonPath.Get(jsonRequest, jsonPath)
	if result.Exists() {
		return result.String(), nil
	}

	return "", mapper.NewErrValueNotFound(jsonPath)
}

func (d dynamicValueService) getResponseValueByJsonPath(jsonPath string, history *vo.History) (string, error) {
	jsonPath = strings.Replace(jsonPath, "responses.", "", 1)

	jsonResponse, err := history.Map()
	if checker.NonNil(err) {
		return "", err
	}

	result := d.jsonPath.Get(jsonResponse, jsonPath)
	if result.Exists() {
		return result.String(), nil
	}

	return "", mapper.NewErrValueNotFound(jsonPath)
}
