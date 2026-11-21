package handler

import (
	"net/http"

	"github.com/chihqiang/ten-go/core/logger"
	"github.com/chihqiang/ten-go/rest"

	"chihqiang/dskpanel/logic"
	"chihqiang/dskpanel/svc"
)

// AuthHandler 登录鉴权处理器。
type AuthHandler struct {
	ctx *svc.AppContext
}

// NewAuthHandler 创建登录鉴权处理器。
func NewAuthHandler(ctx *svc.AppContext) *AuthHandler {
	return &AuthHandler{ctx: ctx}
}

// Login 登录：校验账号密码，签发 token。
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req logic.LoginRequest
	if err := rest.Bind(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}

	result, err := h.ctx.AuthLogic.Login(&req)
	if err != nil {
		logger.Infof("login failed for %s: %v", req.Username, err)
		rest.ErrorJSON(w, rest.NewCodeError(http.StatusUnauthorized, err.Error()))
		return
	}

	rest.OkJSON(w, result)
}
