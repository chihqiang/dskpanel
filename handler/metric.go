package handler

import (
	"net/http"

	"github.com/chihqiang/ten-go/rest"

	"chihqiang/dskpanel/svc"
)

// MetricHandler 指标查询处理器。
type MetricHandler struct {
	ctx *svc.AppContext
}

// NewMetricHandler 创建指标查询处理器。
func NewMetricHandler(ctx *svc.AppContext) *MetricHandler {
	return &MetricHandler{ctx: ctx}
}

// ListNodeMetrics 查询节点指标。
// query: type=docker|swarm|k8s（默认 docker），limit=条数（默认 100）。
func (h *MetricHandler) ListNodeMetrics(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type  string `form:"type,default=docker"`
		Limit int    `form:"limit,default=100"`
	}
	if err := rest.BindQuery(r, &req); err != nil {
		_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.StatusOf(err), err.Error()))
		return
	}
	items, err := h.ctx.MetricLogic.ListNodeMetrics(r.Context(), req.Type, req.Limit)
	if err != nil {
		rest.ErrorJSON(w, rest.NewCodeError(http.StatusInternalServerError, err.Error()))
		return
	}
	rest.OkJSON(w, items)
}
