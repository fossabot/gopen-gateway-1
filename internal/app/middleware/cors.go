package middleware

import (
	"github.com/GabrielHCataldo/go-helper/helper"
	"github.com/GabrielHCataldo/gopen-gateway/internal/app/util"
	"github.com/GabrielHCataldo/gopen-gateway/internal/domain/model/consts"
	"github.com/GabrielHCataldo/gopen-gateway/internal/domain/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

type cors struct {
	securityCorsService service.SecurityCors
}

type SecurityCors interface {
	Do(ctx *gin.Context)
}

func NewSecurityCors(securityCorsService service.SecurityCors) SecurityCors {
	return cors{
		securityCorsService: securityCorsService,
	}
}

func (c cors) Do(ctx *gin.Context) {
	// chamamos o domínio para validar se o ip de origem é permitida a partir do objeto de valor fornecido
	if err := c.securityCorsService.ValidateOrigins(ctx.GetHeader(consts.XForwardedFor)); helper.IsNotNil(err) {
		util.RespondCodeWithError(ctx, http.StatusForbidden, err)
		return
	}
	// chamamos o domínio para validar se o method é permitida a partir do objeto de valor fornecido
	if err := c.securityCorsService.ValidateMethods(ctx.Request.Method); helper.IsNotNil(err) {
		util.RespondCodeWithError(ctx, http.StatusForbidden, err)
		return
	}
	// chamamos o domínio para validar se o headers fornecido estão permitidas a partir do objeto de valor fornecido
	if err := c.securityCorsService.ValidateHeaders(ctx.Request.Header); helper.IsNotNil(err) {
		util.RespondCodeWithError(ctx, http.StatusForbidden, err)
		return
	}

	// se tudo ocorreu bem seguimos para o próximo
	ctx.Next()
}
