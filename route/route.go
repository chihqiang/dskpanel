package route

import (
	"net/http"

	"github.com/chihqiang/infra-go/httpx"

	"chihqiang/dskpanel/handler"
	"chihqiang/dskpanel/middleware"
	"chihqiang/dskpanel/svc"
)

// Register 注册所有路由与中间件。
//
// 路由规划：
//   - POST /api/v1/auth/login                登录（免鉴权）
//   - GET  /api/v1/ping                      健康检查（免鉴权）
//   - GET  /api/v1/docker/detect             本机 Docker 检测（需鉴权）
//   - GET/POST /api/v1/containers            容器列表/创建
//   - GET/POST/DELETE /api/v1/containers/{id}  容器详情/启停/重启/删除
//   - GET/POST/DELETE /api/v1/images         镜像列表/拉取/删除/Tag
//   - GET/POST/DELETE /api/v1/networks       网络列表/创建/删除
//   - GET/POST/DELETE /api/v1/volumes        卷列表/创建/删除
func Register(server *httpx.Server, ctx *svc.AppContext) {
	// 全局中间件。
	server.Use(httpx.WithRequestID())
	server.Use(httpx.WithLogger())
	server.Use(httpx.WithRecovery())
	server.Use(httpx.WithCors("*"))

	authHandler := handler.NewAuthHandler(ctx)
	dockerHandler := handler.NewDockerHandler(ctx)
	containerHandler := handler.NewContainerHandler(ctx)
	terminalHandler := handler.NewTerminalHandler(ctx)
	imageHandler := handler.NewImageHandler(ctx)
	networkHandler := handler.NewNetworkHandler(ctx)
	volumeHandler := handler.NewVolumeHandler(ctx)
	composeHandler := handler.NewComposeHandler(ctx)
	metricHandler := handler.NewMetricHandler(ctx)
	swarmHandler := handler.NewSwarmHandler(ctx)
	k8sHandler := handler.NewK8sHandler(ctx)

	// 公开路由。
	server.AddRoute(httpx.Route{
		Method:  http.MethodPost,
		Path:    "/api/v1/auth/login",
		Handler: authHandler.Login,
	})
	server.AddRoute(httpx.Route{
		Method: http.MethodGet,
		Path:   "/api/v1/ping",
		Handler: func(w http.ResponseWriter, _ *http.Request) {
			httpx.OkJSON(w, "pong")
		},
	})

	// 受保护路由（需登录）。
	authed := server.Group("/api/v1", middleware.Auth(ctx.AuthLogic))

	authed.AddRoute(httpx.Route{
		Method:  http.MethodGet,
		Path:    "/docker/detect",
		Handler: dockerHandler.Detect,
	})
	authed.AddRoute(httpx.Route{
		Method:  http.MethodGet,
		Path:    "/docker/overview",
		Handler: dockerHandler.Overview,
	})
	authed.AddRoute(httpx.Route{
		Method:  http.MethodGet,
		Path:    "/docker/info",
		Handler: dockerHandler.Info,
	})
	authed.AddRoute(httpx.Route{
		Method:  http.MethodPost,
		Path:    "/docker/prune",
		Handler: dockerHandler.PruneAll,
	})
	authed.AddRoute(httpx.Route{
		Method:  http.MethodGet,
		Path:    "/docker/events",
		Handler: dockerHandler.Events,
	})

	// 容器。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/containers", Handler: containerHandler.List})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/containers", Handler: containerHandler.Create})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/containers/{id}", Handler: containerHandler.Inspect})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/containers/{id}/inspect-raw", Handler: containerHandler.InspectRaw})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/containers/{id}/start", Handler: containerHandler.Start})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/containers/{id}/stop", Handler: containerHandler.Stop})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/containers/{id}/restart", Handler: containerHandler.Restart})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/containers/{id}/rename", Handler: containerHandler.Rename})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/containers/{id}/commit", Handler: containerHandler.Commit})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/containers/{id}/update", Handler: containerHandler.Update})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/containers/{id}/pause", Handler: containerHandler.Pause})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/containers/{id}/unpause", Handler: containerHandler.Unpause})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/containers/{id}/export", Handler: containerHandler.Export})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/containers/{id}/stats", Handler: containerHandler.Stats})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/containers/batch", Handler: containerHandler.Batch})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/containers/{id}/top", Handler: containerHandler.Top})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/containers/{id}/exec", Handler: containerHandler.Exec})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/containers/{id}", Handler: containerHandler.Remove})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/containers/{id}/logs", Handler: containerHandler.Logs})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/containers/{id}/logs/stream", Handler: containerHandler.LogsStream})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/containers/{id}/terminal", Handler: terminalHandler.Attach})

	// 镜像。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/images", Handler: imageHandler.List})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/images/pull", Handler: imageHandler.Pull})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/images/push", Handler: imageHandler.Push})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/images/tag", Handler: imageHandler.Tag})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/images/prune", Handler: imageHandler.Prune})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/images/remove", Handler: imageHandler.RemoveBatch})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/images/export", Handler: imageHandler.Export})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/images/import", Handler: imageHandler.Import})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/images/disk-usage", Handler: imageHandler.DiskUsage})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/images/{id}", Handler: imageHandler.Inspect})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/images/{id}", Handler: imageHandler.Remove})

	// 网络。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/networks", Handler: networkHandler.List})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/networks", Handler: networkHandler.Create})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/networks/{id}", Handler: networkHandler.Inspect})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/networks/{id}/connect", Handler: networkHandler.ConnectContainer})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/networks/{id}/disconnect", Handler: networkHandler.DisconnectContainer})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/networks/prune", Handler: networkHandler.Prune})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/networks/{id}", Handler: networkHandler.Remove})

	// 卷。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/volumes", Handler: volumeHandler.List})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/volumes", Handler: volumeHandler.Create})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/volumes/{name}", Handler: volumeHandler.Inspect})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/volumes/prune", Handler: volumeHandler.Prune})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/volumes/{name}", Handler: volumeHandler.Remove})

	// Compose 编排透传。
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/compose/validate", Handler: composeHandler.Validate})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/compose/deploy", Handler: composeHandler.Deploy})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/compose/deploy/stream", Handler: composeHandler.DeployStream})
	// Compose 项目管理。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/compose/projects", Handler: composeHandler.ListProjects})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/compose/projects/{name}/ps", Handler: composeHandler.ProjectPs})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/compose/projects/{name}/start", Handler: composeHandler.ProjectStart})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/compose/projects/{name}/stop", Handler: composeHandler.ProjectStop})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/compose/projects/{name}/restart", Handler: composeHandler.ProjectRestart})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/compose/projects/{name}/down", Handler: composeHandler.ProjectDown})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/compose/projects/{name}/logs", Handler: composeHandler.ProjectLogs})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/compose/projects/{name}/config", Handler: composeHandler.ProjectConfig})

	// 指标查询。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/metrics/nodes", Handler: metricHandler.ListNodeMetrics})

	// Swarm 集群管理（连接目标由 config.yaml 的 swarm 段配置）。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/swarm/detect", Handler: swarmHandler.Detect})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/swarm/overview", Handler: swarmHandler.Overview})
	// 节点。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/swarm/nodes", Handler: swarmHandler.ListNodes})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/swarm/nodes/{id}", Handler: swarmHandler.InspectNode})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/swarm/nodes/{id}/availability", Handler: swarmHandler.SetNodeAvailability})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/swarm/nodes/{id}", Handler: swarmHandler.RemoveNode})
	// 服务。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/swarm/services", Handler: swarmHandler.ListServices})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/swarm/services/{id}", Handler: swarmHandler.InspectService})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/swarm/services/{id}/resources", Handler: swarmHandler.ServiceResources})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/swarm/services", Handler: swarmHandler.CreateService})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/swarm/services/{id}", Handler: swarmHandler.UpdateService})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/swarm/services/{id}/scale", Handler: swarmHandler.ScaleService})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/swarm/services/{id}/rollback", Handler: swarmHandler.RollbackService})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/swarm/services/{id}/force-update", Handler: swarmHandler.ForceUpdateService})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/swarm/services/{id}", Handler: swarmHandler.RemoveService})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/swarm/services/{id}/logs", Handler: swarmHandler.ServiceLogs})
	// 任务。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/swarm/tasks", Handler: swarmHandler.ListTasks})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/swarm/tasks/{id}/logs", Handler: swarmHandler.TaskLogs})
	// 网络（供服务表单选择 overlay 网络）。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/swarm/networks", Handler: swarmHandler.ListNetworks})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/swarm/networks", Handler: swarmHandler.CreateNetwork})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/swarm/networks/{id}", Handler: swarmHandler.InspectNetwork})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/swarm/networks/{id}", Handler: swarmHandler.RemoveNetwork})
	// 加入令牌。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/swarm/join-tokens", Handler: swarmHandler.GetJoinTokens})
	// 集群镜像列表（服务创建表单选择镜像）。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/swarm/images", Handler: swarmHandler.ListImages})
	// Secret。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/swarm/secrets", Handler: swarmHandler.ListSecrets})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/swarm/secrets/{id}", Handler: swarmHandler.InspectSecret})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/swarm/secrets", Handler: swarmHandler.CreateSecret})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/swarm/secrets/{id}", Handler: swarmHandler.RemoveSecret})
	// Config。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/swarm/configs", Handler: swarmHandler.ListConfigs})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/swarm/configs/{id}", Handler: swarmHandler.InspectConfig})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/swarm/configs", Handler: swarmHandler.CreateConfig})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/swarm/configs/{id}", Handler: swarmHandler.RemoveConfig})

	// Kubernetes 集群管理。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/detect", Handler: k8sHandler.Detect})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/overview", Handler: k8sHandler.Overview})
	// 事件。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/events", Handler: k8sHandler.ListEvents})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/events/{kind}/{name}", Handler: k8sHandler.ListEventsForResource})
	// 节点。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/nodes", Handler: k8sHandler.ListNodes})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/nodes/{name}", Handler: k8sHandler.InspectNode})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/nodes/{name}/usage", Handler: k8sHandler.NodeUsage})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/k8s/nodes/{name}/cordon", Handler: k8sHandler.CordonNode})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/k8s/nodes/{name}/uncordon", Handler: k8sHandler.UncordonNode})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/k8s/nodes/{name}/drain", Handler: k8sHandler.DrainNode})
	// 命名空间。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/namespaces", Handler: k8sHandler.ListNamespaces})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/namespaces/{name}", Handler: k8sHandler.InspectNamespace})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/k8s/namespaces/{name}", Handler: k8sHandler.DeleteNamespace})
	// Pod。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/pods", Handler: k8sHandler.ListPods})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/pods/{name}", Handler: k8sHandler.InspectPod})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/k8s/pods/{name}", Handler: k8sHandler.DeletePod})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/pods/{name}/logs", Handler: k8sHandler.PodLogs})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/k8s/pods/{name}/exec", Handler: k8sHandler.ExecPod})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/pods/{name}/terminal", Handler: k8sHandler.PodTerminal})
	// Deployment。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/deployments", Handler: k8sHandler.ListDeployments})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/deployments/{name}", Handler: k8sHandler.InspectDeployment})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/k8s/deployments/{name}", Handler: k8sHandler.DeleteDeployment})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/k8s/deployments/{name}/scale", Handler: k8sHandler.ScaleDeployment})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/k8s/deployments/{name}/restart", Handler: k8sHandler.RestartDeployment})
	// StatefulSet。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/statefulsets", Handler: k8sHandler.ListStatefulSets})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/statefulsets/{name}", Handler: k8sHandler.InspectStatefulSet})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/k8s/statefulsets/{name}", Handler: k8sHandler.DeleteStatefulSet})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/k8s/statefulsets/{name}/scale", Handler: k8sHandler.ScaleStatefulSet})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/k8s/statefulsets/{name}/restart", Handler: k8sHandler.RestartStatefulSet})
	// DaemonSet。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/daemonsets", Handler: k8sHandler.ListDaemonSets})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/daemonsets/{name}", Handler: k8sHandler.InspectDaemonSet})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/k8s/daemonsets/{name}", Handler: k8sHandler.DeleteDaemonSet})
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/k8s/daemonsets/{name}/restart", Handler: k8sHandler.RestartDaemonSet})
	// Job。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/jobs", Handler: k8sHandler.ListJobs})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/jobs/{name}", Handler: k8sHandler.InspectJob})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/k8s/jobs/{name}", Handler: k8sHandler.DeleteJob})
	// CronJob。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/cronjobs", Handler: k8sHandler.ListCronJobs})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/cronjobs/{name}", Handler: k8sHandler.InspectCronJob})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/k8s/cronjobs/{name}", Handler: k8sHandler.DeleteCronJob})
	// Service。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/services", Handler: k8sHandler.ListServices})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/services/{name}", Handler: k8sHandler.InspectService})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/k8s/services/{name}", Handler: k8sHandler.DeleteService})
	// Ingress。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/ingresses", Handler: k8sHandler.ListIngresses})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/ingresses/{name}", Handler: k8sHandler.InspectIngress})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/k8s/ingresses/{name}", Handler: k8sHandler.DeleteIngress})
	// ConfigMap。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/configmaps", Handler: k8sHandler.ListConfigMaps})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/configmaps/{name}", Handler: k8sHandler.InspectConfigMap})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/k8s/configmaps/{name}", Handler: k8sHandler.DeleteConfigMap})
	// Secret。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/secrets", Handler: k8sHandler.ListSecrets})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/secrets/{name}", Handler: k8sHandler.InspectSecret})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/k8s/secrets/{name}", Handler: k8sHandler.DeleteSecret})
	// PVC。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/pvcs", Handler: k8sHandler.ListPVCs})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/pvcs/{name}", Handler: k8sHandler.InspectPVC})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/k8s/pvcs/{name}", Handler: k8sHandler.DeletePVC})
	// StorageClass。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/storageclasses", Handler: k8sHandler.ListStorageClasses})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/storageclasses/{name}", Handler: k8sHandler.InspectStorageClass})
	// RBAC - Role。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/roles", Handler: k8sHandler.ListRoles})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/roles/{name}", Handler: k8sHandler.InspectRole})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/k8s/roles/{name}", Handler: k8sHandler.DeleteRole})
	// RBAC - ClusterRole。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/clusterroles", Handler: k8sHandler.ListClusterRoles})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/clusterroles/{name}", Handler: k8sHandler.InspectClusterRole})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/k8s/clusterroles/{name}", Handler: k8sHandler.DeleteClusterRole})
	// RBAC - RoleBinding。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/rolebindings", Handler: k8sHandler.ListRoleBindings})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/rolebindings/{name}", Handler: k8sHandler.InspectRoleBinding})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/k8s/rolebindings/{name}", Handler: k8sHandler.DeleteRoleBinding})
	// RBAC - ClusterRoleBinding。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/clusterrolebindings", Handler: k8sHandler.ListClusterRoleBindings})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/clusterrolebindings/{name}", Handler: k8sHandler.InspectClusterRoleBinding})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/k8s/clusterrolebindings/{name}", Handler: k8sHandler.DeleteClusterRoleBinding})
	// HPA（自动伸缩）。
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/hpas", Handler: k8sHandler.ListHPAs})
	authed.AddRoute(httpx.Route{Method: http.MethodGet, Path: "/k8s/hpas/{name}", Handler: k8sHandler.InspectHPA})
	authed.AddRoute(httpx.Route{Method: http.MethodDelete, Path: "/k8s/hpas/{name}", Handler: k8sHandler.DeleteHPA})
	// YAML 透传（kubectl apply 语义，支持多文档 YAML）。
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/k8s/apply", Handler: k8sHandler.ApplyYAML})
	// YAML 删除（kubectl delete -f 语义，支持多文档 YAML）。
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/k8s/delete", Handler: k8sHandler.DeleteYAML})
	// YAML 验证（kubectl apply --dry-run=server 语义）。
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/k8s/dryrun", Handler: k8sHandler.DryRunYAML})
	// 批量删除资源（支持跨类型、跨命名空间）。
	authed.AddRoute(httpx.Route{Method: http.MethodPost, Path: "/k8s/delete/resources", Handler: k8sHandler.DeleteResources})
}
