/**
 * Kubernetes 资源 YAML 模板库（kubectl apply 语义）。
 *
 * 按资源类型分组导出，供各页面按「入口」选择对应的模板集：
 * - k8sPodTemplates      → Pod 列表页
 * - k8sWorkloadTemplates → 工作负载页（Deployment / StatefulSet / DaemonSet）
 * - k8sServiceTemplates  → 服务页（Service / Ingress）
 * - k8sConfigTemplates   → 配置页（ConfigMap / Secret）
 * - k8sNodeTemplates     → 节点页
 * - k8sTemplates         → 通用（默认）
 */

import type { YamlTemplate } from './serviceSpec'

// ──────────────────────────────────────────────
// Pod
// ──────────────────────────────────────────────

/** 单容器 Pod。 */
const singlePod: YamlTemplate = {
  name: '单容器 Pod',
  desc: 'busybox 执行一次性命令后退出',
  yaml: `apiVersion: v1
kind: Pod
metadata:
  name: busybox
  namespace: default
spec:
  restartPolicy: Never
  containers:
    - name: busybox
      image: busybox:latest
      command: ["sh", "-c", "echo hello; sleep 30"]
`,
}

/** 多容器 Pod（主容器 + sidecar）。 */
const multiContainerPod: YamlTemplate = {
  name: '多容器 Pod',
  desc: 'nginx + sidecar 日志容器',
  yaml: `apiVersion: v1
kind: Pod
metadata:
  name: multi-container
  namespace: default
spec:
  containers:
    - name: nginx
      image: nginx:latest
      ports:
        - containerPort: 80
    - name: sidecar
      image: busybox:latest
      command: ["sh", "-c", "tail -f /var/log/nginx/access.log"]
      volumeMounts:
        - name: logs
          mountPath: /var/log/nginx
  volumes:
    - name: logs
      emptyDir: {}
`,
}

/** Pod 模板集。 */
export const k8sPodTemplates: YamlTemplate[] = [singlePod, multiContainerPod]

// ──────────────────────────────────────────────
// 工作负载
// ──────────────────────────────────────────────

/** Nginx Deployment。 */
const nginxDeployment: YamlTemplate = {
  name: 'Nginx Deployment',
  desc: '单副本 Deployment + 标签选择器',
  yaml: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: nginx:latest
          ports:
            - containerPort: 80
`,
}

/** 带资源限制 + 健康检查的 Deployment。 */
const resourceful: YamlTemplate = {
  name: '资源限制 + 健康检查',
  desc: 'requests/limits + liveness/readiness',
  yaml: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  namespace: default
spec:
  replicas: 2
  selector:
    matchLabels:
      app: app
  template:
    metadata:
      labels:
        app: app
    spec:
      containers:
        - name: app
          image: nginx:latest
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 512Mi
          livenessProbe:
            httpGet:
              path: /
              port: 80
          readinessProbe:
            httpGet:
              path: /
              port: 80
`,
}

/** Redis StatefulSet + Headless Service。 */
const redisStatefulSet: YamlTemplate = {
  name: 'Redis StatefulSet',
  desc: '有状态服务 + Headless Service',
  yaml: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: redis
  namespace: default
spec:
  serviceName: redis-headless
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
        - name: redis
          image: redis:7-alpine
          ports:
            - containerPort: 6379
---
apiVersion: v1
kind: Service
metadata:
  name: redis-headless
  namespace: default
spec:
  clusterIP: None
  selector:
    app: redis
  ports:
    - port: 6379
      targetPort: 6379
`,
}

/** 日志收集 DaemonSet。 */
const logDaemonSet: YamlTemplate = {
  name: '日志收集 DaemonSet',
  desc: '每个节点运行一个日志采集 Pod',
  yaml: `apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluentd
  namespace: default
spec:
  selector:
    matchLabels:
      app: fluentd
  template:
    metadata:
      labels:
        app: fluentd
    spec:
      containers:
        - name: fluentd
          image: fluent/fluentd:latest
`,
}

/** 工作负载模板集。 */
export const k8sWorkloadTemplates: YamlTemplate[] = [
  nginxDeployment,
  resourceful,
  redisStatefulSet,
  logDaemonSet,
]

// ──────────────────────────────────────────────
// Service / Ingress
// ──────────────────────────────────────────────

/** ClusterIP Service。 */
const clusterIpService: YamlTemplate = {
  name: 'ClusterIP Service',
  desc: '集群内部访问',
  yaml: `apiVersion: v1
kind: Service
metadata:
  name: my-service
  namespace: default
spec:
  type: ClusterIP
  selector:
    app: my-app
  ports:
    - port: 80
      targetPort: 8080
`,
}

/** NodePort Service。 */
const nodePortService: YamlTemplate = {
  name: 'NodePort Service',
  desc: '节点端口对外暴露',
  yaml: `apiVersion: v1
kind: Service
metadata:
  name: web
  namespace: default
spec:
  type: NodePort
  selector:
    app: web
  ports:
    - port: 80
      targetPort: 80
      nodePort: 30080
`,
}

/** Ingress 域名路由。 */
const ingress: YamlTemplate = {
  name: 'Ingress',
  desc: '域名路由转发到 Service',
  yaml: `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example
  namespace: default
spec:
  ingressClassName: nginx
  rules:
    - host: example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: web
                port:
                  number: 80
`,
}

/** Service / Ingress 模板集。 */
export const k8sServiceTemplates: YamlTemplate[] = [
  clusterIpService,
  nodePortService,
  ingress,
]

// ──────────────────────────────────────────────
// ConfigMap / Secret
// ──────────────────────────────────────────────

/** ConfigMap。 */
const configMap: YamlTemplate = {
  name: 'ConfigMap',
  desc: '应用配置数据',
  yaml: `apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  namespace: default
data:
  APP_ENV: production
  LOG_LEVEL: info
  DATABASE_URL: postgres://db:5432/app
`,
}

/** Secret（Base64 编码）。 */
const secret: YamlTemplate = {
  name: 'Secret',
  desc: 'Base64 编码的敏感数据',
  yaml: `apiVersion: v1
kind: Secret
metadata:
  name: app-secret
  namespace: default
type: Opaque
data:
  username: YWRtaW4=        # admin
  password: c2VjcmV0MTIz    # secret123
`,
}

/** ConfigMap / Secret 模板集。 */
export const k8sConfigTemplates: YamlTemplate[] = [configMap, secret]

// ──────────────────────────────────────────────
// Node
// ──────────────────────────────────────────────

/** 为节点添加标签。 */
const nodeLabel: YamlTemplate = {
  name: '节点标签',
  desc: '为节点添加标签（调度用）',
  yaml: `apiVersion: v1
kind: Node
metadata:
  name: node-01
  labels:
    disktype: ssd
    environment: production
`,
}

/** Node 模板集。 */
export const k8sNodeTemplates: YamlTemplate[] = [nodeLabel]

// ──────────────────────────────────────────────
// 通用（默认）
// ──────────────────────────────────────────────

/** 通用模板集（默认兜底）。 */
export const k8sTemplates: YamlTemplate[] = [
  nginxDeployment,
  resourceful,
  nodePortService,
  configMap,
]
