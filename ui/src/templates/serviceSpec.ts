/** Swarm ServiceSpec YAML 模板。 */

export interface YamlTemplate {
  name: string
  /** 一句话说明。 */
  desc: string
  yaml: string
}

/** Nginx Web 服务（副本 + 端口）。 */
const nginxWeb: YamlTemplate = {
  name: 'Nginx Web 服务',
  desc: 'Replicated 副本 + 8080 端口映射',
  yaml: `Name: my-web
Labels:
  env: prod
  app: web
Mode:
  Replicated:
    Replicas: 2
TaskTemplate:
  ContainerSpec:
    Image: nginx:alpine
    Env:
      - TZ=Asia/Shanghai
    Args:
      - nginx
      - -g
      - daemon off;
  RestartPolicy:
    Condition: any
  Resources:
    Limits:
      NanoCPUs: 500000000
      MemoryBytes: 536870912
  Placement:
    Constraints:
      - node.role == manager
UpdateConfig:
  Parallelism: 2
  Delay: 5000000000
  FailureAction: pause
EndpointSpec:
  Mode: vip
  Ports:
    - TargetPort: 80
      PublishedPort: 8080
      Protocol: tcp
      PublishMode: ingress
`,
}

/** Redis 缓存（无端口暴露，内部访问）。 */
const redisCache: YamlTemplate = {
  name: 'Redis 缓存',
  desc: '单副本 + 资源限制 + 重启策略',
  yaml: `Name: redis-cache
Mode:
  Replicated:
    Replicas: 1
TaskTemplate:
  ContainerSpec:
    Image: redis:7-alpine
    Command:
      - redis-server
      - --appendonly yes
    Env:
      - TZ=Asia/Shanghai
  RestartPolicy:
    Condition: any
  Resources:
    Limits:
      NanoCPUs: 500000000
      MemoryBytes: 268435456
`,
}

/** 带 Secret 挂载的服务。 */
const withSecret: YamlTemplate = {
  name: '带 Secret 挂载',
  desc: '演示 Secret 文件挂载（需先创建 Secret）',
  yaml: `Name: app-with-secret
Mode:
  Replicated:
    Replicas: 1
TaskTemplate:
  ContainerSpec:
    Image: nginx:alpine
    Secrets:
      - SecretName: db_password
        File:
          Name: db_password
          UID: "0"
          GID: "0"
          Mode: 0440
  RestartPolicy:
    Condition: any
`,
}

/** 带 Config 挂载 + 滚动更新。 */
const withConfig: YamlTemplate = {
  name: 'Config + 滚动更新',
  desc: '演示 Config 挂载与滚动更新策略',
  yaml: `Name: app-with-config
Mode:
  Replicated:
    Replicas: 3
TaskTemplate:
  ContainerSpec:
    Image: nginx:alpine
    Configs:
      - ConfigName: nginx_conf
        File:
          Name: /etc/nginx/nginx.conf
  RestartPolicy:
    Condition: any
UpdateConfig:
  Parallelism: 1
  Delay: 10000000000
  FailureAction: rollback
  Monitor: 30000000000
  Order: start-first
`,
}

/** 全局模式（每节点一个）。 */
const globalDaemon: YamlTemplate = {
  name: '全局模式服务',
  desc: 'Global 模式：每个节点运行一个',
  yaml: `Name: node-exporter
Mode:
  Global: {}
TaskTemplate:
  ContainerSpec:
    Image: prom/node-exporter:latest
    Command:
      - --path.procfs=/host/proc
      - --path.sysfs=/host/sys
    Mounts:
      - Type: bind
        Source: /proc
        Target: /host/proc
      - Type: bind
        Source: /sys
        Target: /host/sys
      - Type: bind
        Source: /
        Target: /rootfs
  RestartPolicy:
    Condition: any
Placement:
  Constraints:
    - node.role == manager
`,
}

/** 常用 ServiceSpec 模板集合。 */
export const serviceSpecTemplates: YamlTemplate[] = [
  nginxWeb,
  redisCache,
  withSecret,
  withConfig,
  globalDaemon,
]
