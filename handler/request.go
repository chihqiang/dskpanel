package handler

import (
	"net/http"

	"github.com/chihqiang/ten-go/core/cast"
)

// queryValue 读取查询参数并转换为 T；参数缺失、为空或转换失败时返回 def。
func queryValue[T any](r *http.Request, key string, def ...T) T {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return defaultOf(def)
	}
	v, err := cast.ToE[T](raw)
	if err != nil {
		return defaultOf(def)
	}
	return v
}

func defaultOf[T any](def []T) T {
	var zero T
	if len(def) > 0 {
		return def[0]
	}
	return zero
}
