/** 常用 YAML 模板库：集中维护，各组件引入复用。 */

export type { YamlTemplate } from './serviceSpec'
export { serviceSpecTemplates } from './serviceSpec'
export { composeTemplates } from './compose'
export {
  k8sTemplates,
  k8sPodTemplates,
  k8sWorkloadTemplates,
  k8sServiceTemplates,
  k8sConfigTemplates,
  k8sNodeTemplates,
} from './k8s'
