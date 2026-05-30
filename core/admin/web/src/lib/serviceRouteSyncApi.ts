import { api } from '../api/client'
import { parseModuleDoc, stringifyModuleDoc } from './ingressModuleForms'
import { servicesFromDoc } from './services'
import {
  detectServiceRouteConflicts,
  mergeServicesDocsIntoRulesDoc,
  type ServiceRouteConflict,
  type ServiceRouteSyncResolutionMap,
  type ServiceRouteSyncResult,
} from './serviceRouteSync'

export type ServicesPersistPlan = {
  content: string
  sync: ServiceRouteSyncResult
  conflicts: ServiceRouteConflict[]
}

/** Build full ingress YAML after services module change + optional route sync. */
export async function buildServicesPersistContent(
  baseContent: string,
  servicesDoc: Record<string, unknown>,
  prevServicesDoc: Record<string, unknown>,
  resolutions: ServiceRouteSyncResolutionMap = {},
): Promise<ServicesPersistPlan> {
  const modules = await api.configModules(baseContent)
  const rulesMod = modules.find((m) => m.id === 'rules')
  const rulesDoc = parseModuleDoc(rulesMod?.yaml ?? '')

  const conflicts = detectServiceRouteConflicts(
    rulesDoc,
    servicesFromDoc(prevServicesDoc),
    servicesFromDoc(servicesDoc),
  )

  const { rulesDoc: syncedRules, sync } = mergeServicesDocsIntoRulesDoc(
    rulesDoc,
    servicesDoc,
    prevServicesDoc,
    resolutions,
  )

  const servicesYAML = stringifyModuleDoc(servicesDoc)
  let content = (await api.mergeConfigModule(baseContent, 'services', servicesYAML)).content

  if (sync.touched > 0) {
    const rulesYAML = stringifyModuleDoc(syncedRules)
    content = (await api.mergeConfigModule(content, 'rules', rulesYAML)).content
  }

  return { content, sync, conflicts }
}

/** @deprecated use buildServicesPersistContent */
export async function mergeServicesModuleWithRouteSync(
  baseContent: string,
  servicesDoc: Record<string, unknown>,
  prevServicesDoc: Record<string, unknown>,
  resolutions: ServiceRouteSyncResolutionMap = {},
): Promise<{ content: string; sync: ServiceRouteSyncResult }> {
  const plan = await buildServicesPersistContent(baseContent, servicesDoc, prevServicesDoc, resolutions)
  return { content: plan.content, sync: plan.sync }
}
