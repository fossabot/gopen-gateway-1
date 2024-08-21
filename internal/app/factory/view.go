package factory

import (
	"github.com/tech4works/checker"
	"github.com/tech4works/gopen-gateway/internal/app/model/dto"
)

func BuildSettingView(gopen dto.Gopen) dto.SettingView {
	copied := gopen
	copied.Store = nil

	return dto.SettingView{
		Version:      "v1.0.0",
		VersionDate:  "05/10/2024",
		Founder:      "Gabriel Cataldo",
		Contributors: 1,
		Endpoints:    countEndpoints(gopen),
		Middlewares:  countMiddlewares(gopen),
		Backends:     countBackends(gopen),
		Setting:      copied,
	}
}

func countEndpoints(gopen dto.Gopen) int {
	return len(gopen.Endpoints)
}

func countMiddlewares(gopen dto.Gopen) int {
	if checker.NonNil(gopen.Middlewares) {
		return len(gopen.Middlewares)
	}
	return 0
}

func countBackends(gopen dto.Gopen) (count int) {
	for _, endpoint := range gopen.Endpoints {
		count += len(endpoint.Beforewares) + len(endpoint.Backends) + len(endpoint.Afterwares)
	}
	return count
}
