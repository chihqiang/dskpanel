/** Compose 文件 YAML 模板。 */

export interface YamlTemplate {
  name: string
  /** 一句话说明。 */
  desc: string
  yaml: string
}

/** 单服务 Web。 */
const singleWeb: YamlTemplate = {
  name: '单服务 Web',
  desc: 'nginx + 端口映射 + 卷挂载',
  yaml: `services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
    volumes:
      - ./html:/usr/share/nginx/html:ro
    environment:
      - TZ=Asia/Shanghai
    restart: unless-stopped
`,
}

/** Web + DB 多服务。 */
const webDb: YamlTemplate = {
  name: 'Web + DB',
  desc: '多服务 + 依赖 + 命名卷',
  yaml: `services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
    depends_on:
      - db
    environment:
      - DB_HOST=db
    restart: unless-stopped
  db:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=secret
      - MYSQL_DATABASE=app
    volumes:
      - dbdata:/var/lib/mysql
    restart: unless-stopped
volumes:
  dbdata:
`,
}

/** 带自定义网络。 */
const withNetwork: YamlTemplate = {
  name: '自定义网络',
  desc: '前后端分区网络',
  yaml: `services:
  frontend:
    image: nginx:latest
    ports:
      - "8080:80"
    networks:
      - frontend
      - backend
  backend:
    image: node:22-alpine
    networks:
      - backend
networks:
  frontend:
  backend:
`,
}

/** 环境变量 + 命令 + 资源限制。 */
const envAndCmd: YamlTemplate = {
  name: 'Env / 命令 / 资源限制',
  desc: '演示常用服务配置',
  yaml: `services:
  api:
    image: node:22-alpine
    command: ["node", "server.js"]
    environment:
      - NODE_ENV=production
      - PORT=3000
    ports:
      - "3000:3000"
    deploy:
      resources:
        limits:
          cpus: "0.5"
          memory: 512M
    restart: always
`,
}

/** 常用 Compose 模板集合。 */
export const composeTemplates: YamlTemplate[] = [singleWeb, webDb, withNetwork, envAndCmd]
