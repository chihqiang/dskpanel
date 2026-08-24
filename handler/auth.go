package handler

import (
	"net/http"

	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/logger"

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
	if err := httpx.MustBindJSON(w, r, &req); err != nil {
		return // 已自动写 400
	}

	result, err := h.ctx.AuthLogic.Login(&req)
	if err != nil {
		logger.Infof("login failed for %s: %v", req.Username, err)
		httpx.WriteHTTPError(w, http.StatusUnauthorized, err.Error())
		return
	}

	httpx.OkJSON(w, result)
}
