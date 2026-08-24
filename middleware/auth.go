package middleware

import (
	"net/http"
	"strings"

	"github.com/chihqiang/infra-go/httpx"

	"chihqiang/dskpanel/logic"
)

// Auth 登录鉴权中间件。
// 从 Authorization: Bearer <token> 或 query token=<token>（WebSocket 场景）提取 token 校验，失败返回 401。
func Auth(authLogic *logic.AuthLogic) httpx.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				token = r.URL.Query().Get("token")
			}
			if _, err := authLogic.Verify(token); err != nil {
				httpx.WriteHTTPError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			next(w, r)
		}
	}
}

// bearerToken 从 Authorization 头提取 Bearer token。
func bearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return ""
}
