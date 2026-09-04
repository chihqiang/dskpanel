package handler

import (
	"net/http"

	"github.com/chihqiang/infra-go/httpx"

	"chihqiang/dskpanel/logic"
)

// ──────────────────────────────────────────────
// Role
// ──────────────────────────────────────────────

// ListRoles Role 列表（支持 labelSelector / fieldSelector 过滤）。
func (h *K8sHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace     string `form:"namespace"`
		LabelSelector string `form:"labelSelector"`
		FieldSelector string `form:"fieldSelector"`
		Limit         int64  `form:"limit"`
	}
	if err := httpx.MustBindQuery(w, r, &req); err != nil {
		return
	}
	items, err := h.ctx.K8sLogic.ListRolesWithOptions(r.Context(), logic.K8sListOptions{
		Namespace:     req.Namespace,
		LabelSelector: req.LabelSelector,
		FieldSelector: req.FieldSelector,
		Limit:         req.Limit,
	})
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sList(w, r, items)
}

// InspectRole Role 详情（支持 ?format=yaml）。
func (h *K8sHandler) InspectRole(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	role, err := h.ctx.K8sLogic.InspectRole(r.Context(), namespace, r.PathValue("name"))
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sObject(w, r, role)
}

// DeleteRole 删除 Role（NotFound 幂等）。
func (h *K8sHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	err := h.ctx.K8sLogic.DeleteRole(r.Context(), namespace, r.PathValue("name"))
	writeK8sDeleteResult(w, err)
}

// ──────────────────────────────────────────────
// ClusterRole
// ──────────────────────────────────────────────

// ListClusterRoles ClusterRole 列表。
func (h *K8sHandler) ListClusterRoles(w http.ResponseWriter, r *http.Request) {
	items, err := h.ctx.K8sLogic.ListClusterRoles(r.Context())
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sList(w, r, items)
}

// InspectClusterRole ClusterRole 详情（支持 ?format=yaml）。
func (h *K8sHandler) InspectClusterRole(w http.ResponseWriter, r *http.Request) {
	cr, err := h.ctx.K8sLogic.InspectClusterRole(r.Context(), r.PathValue("name"))
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sObject(w, r, cr)
}

// DeleteClusterRole 删除 ClusterRole（NotFound 幂等）。
func (h *K8sHandler) DeleteClusterRole(w http.ResponseWriter, r *http.Request) {
	err := h.ctx.K8sLogic.DeleteClusterRole(r.Context(), r.PathValue("name"))
	writeK8sDeleteResult(w, err)
}

// ──────────────────────────────────────────────
// RoleBinding
// ──────────────────────────────────────────────

// ListRoleBindings RoleBinding 列表。
func (h *K8sHandler) ListRoleBindings(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	items, err := h.ctx.K8sLogic.ListRoleBindings(r.Context(), namespace)
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sList(w, r, items)
}

// InspectRoleBinding RoleBinding 详情（支持 ?format=yaml）。
func (h *K8sHandler) InspectRoleBinding(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	rb, err := h.ctx.K8sLogic.InspectRoleBinding(r.Context(), namespace, r.PathValue("name"))
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sObject(w, r, rb)
}

// DeleteRoleBinding 删除 RoleBinding（NotFound 幂等）。
func (h *K8sHandler) DeleteRoleBinding(w http.ResponseWriter, r *http.Request) {
	namespace := httpx.QueryValue[string](r, "namespace")
	err := h.ctx.K8sLogic.DeleteRoleBinding(r.Context(), namespace, r.PathValue("name"))
	writeK8sDeleteResult(w, err)
}

// ──────────────────────────────────────────────
// ClusterRoleBinding
// ──────────────────────────────────────────────

// ListClusterRoleBindings ClusterRoleBinding 列表。
func (h *K8sHandler) ListClusterRoleBindings(w http.ResponseWriter, r *http.Request) {
	items, err := h.ctx.K8sLogic.ListClusterRoleBindings(r.Context())
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sList(w, r, items)
}

// InspectClusterRoleBinding ClusterRoleBinding 详情（支持 ?format=yaml）。
func (h *K8sHandler) InspectClusterRoleBinding(w http.ResponseWriter, r *http.Request) {
	crb, err := h.ctx.K8sLogic.InspectClusterRoleBinding(r.Context(), r.PathValue("name"))
	if err != nil {
		writeK8sError(w, err)
		return
	}
	writeK8sObject(w, r, crb)
}

// DeleteClusterRoleBinding 删除 ClusterRoleBinding（NotFound 幂等）。
func (h *K8sHandler) DeleteClusterRoleBinding(w http.ResponseWriter, r *http.Request) {
	err := h.ctx.K8sLogic.DeleteClusterRoleBinding(r.Context(), r.PathValue("name"))
	writeK8sDeleteResult(w, err)
}
