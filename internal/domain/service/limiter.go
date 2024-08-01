package service

import (
	"github.com/GabrielHCataldo/go-helper/helper"
	"github.com/GabrielHCataldo/gopen-gateway/internal/domain/mapper"
	"github.com/GabrielHCataldo/gopen-gateway/internal/domain/model/vo"
	timerate "golang.org/x/time/rate"
	"io"
	"net/http"
	"sync"
)

type limiterService struct {
	keys  map[string]*timerate.Limiter
	mutex *sync.RWMutex
}

type Limiter interface {
	AllowRate(request *vo.HTTPRequest, rate vo.Rate) error
	AllowSize(request *vo.HTTPRequest, limiter vo.Limiter) error
}

func NewLimiter() Limiter {
	return &limiterService{
		keys:  map[string]*timerate.Limiter{},
		mutex: &sync.RWMutex{},
	}
}

func (s *limiterService) AllowRate(request *vo.HTTPRequest, rate vo.Rate) (err error) {
	if !rate.HasData() {
		return nil
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	clientIP := request.Header().GetFirst(mapper.XForwardedFor)

	rateLimiter, exists := s.keys[clientIP]
	if !exists {
		rateLimiter = timerate.NewLimiter(timerate.Every(rate.EveryTime()), rate.Capacity())
		s.keys[clientIP] = rateLimiter
	}

	if !rateLimiter.Allow() {
		err = mapper.NewErrTooManyRequests(rate.Capacity(), rate.EveryTime())
	}

	return err
}

func (s *limiterService) AllowSize(request *vo.HTTPRequest, limiter vo.Limiter) error {
	maxHeaderSize := limiter.MaxHeaderSize()
	if helper.IsGreaterThan(request.Header().Size(), maxHeaderSize) {
		return mapper.NewErrHeaderTooLarge(maxHeaderSize.String())
	}

	maxBodySize := limiter.MaxBodySize()
	if helper.ContainsIgnoreCase(request.Header().Get(mapper.ContentType), "multipart/form-data") {
		maxBodySize = limiter.MaxMultipartMemorySize()
	}

	bodyBuffer := request.Body().Buffer()
	readCloser := http.MaxBytesReader(nil, io.NopCloser(bodyBuffer), int64(maxBodySize))

	_, err := io.ReadAll(readCloser)
	if helper.IsNotNil(err) {
		return mapper.NewErrPayloadTooLarge(maxBodySize.String())
	}

	return nil
}
