import { rulesFromDoc } from './configEntities'
import { obj, str, num, bool, arr } from './ingressModuleForms'
import { servicesFromDoc } from './services'

export type ServiceRouteSyncResult = {
  touched: number
  skipped: number
  details: string[]
}

export type ServiceRouteConflict = {
  id: string
  label: string
  serviceName: string
  matchName: string
  routeSummary: string
}

export type ServiceRouteSyncResolution = 'overwrite' | 'keep'

export type ServiceRouteSyncResolutionMap = Record<string, ServiceRouteSyncResolution>

type SyncOp = {
  matchNames: Set<string>
  catalog: Record<string, unknown>
  prevRow: Record<string, unknown>
}

/** Catalog-owned fields copied into rules[].backend.service (route-specific fields preserved). */
export function patchServiceObjectFromCatalog(
  service: Record<string, unknown>,
  catalog: Record<string, unknown>,
): Record<string, unknown> {
  const next: Record<string, unknown> = { ...service }
  next.name = str(catalog.name).trim()
  next.port = num(catalog.port, 8080)

  const protocol = str(catalog.protocol, 'http') || 'http'
  if (protocol !== 'http') next.protocol = protocol
  else delete next.protocol

  const mode = str(catalog.mode)
  if (mode === 'internal' || mode === 'external') next.mode = mode
  else delete next.mode

  const hc = obj(catalog.healthcheck)
  if (bool(hc.enable)) {
    const hcOut: Record<string, unknown> = { enable: true }
    const method = str(hc.method, 'GET') || 'GET'
    if (method !== 'GET') hcOut.method = method
    const path = str(hc.path, '/health') || '/health'
    if (path !== '/health') hcOut.path = path
    const status = arr<number>(hc.status)
    if (status.length > 0) hcOut.status = status
    if (bool(hc.ok)) hcOut.ok = true
    next.healthcheck = hcOut
  } else {
    delete next.healthcheck
  }

  return next
}

export function serviceCatalogCoreFingerprint(service: Record<string, unknown>): string {
  const hc = obj(service.healthcheck)
  return JSON.stringify({
    name: str(service.name).trim(),
    port: num(service.port, 8080),
    protocol: str(service.protocol, 'http') || 'http',
    mode: str(service.mode),
    healthcheck: hc,
  })
}

export function catalogRowToServiceCore(row: Record<string, unknown>): Record<string, unknown> {
  return patchServiceObjectFromCatalog({ name: str(row.name).trim() || 'placeholder' }, row)
}

function catalogCoreFingerprint(row: Record<string, unknown>): string {
  return serviceCatalogCoreFingerprint(catalogRowToServiceCore(row))
}

/** oldName -> newName when the same catalog row was renamed (index-aligned). */
export function buildServiceRenameMap(
  prevCatalog: Record<string, unknown>[],
  nextCatalog: Record<string, unknown>[],
): Map<string, string> {
  const map = new Map<string, string>()
  const len = Math.max(prevCatalog.length, nextCatalog.length)
  for (let i = 0; i < len; i++) {
    const prev = prevCatalog[i]
    const next = nextCatalog[i]
    if (!prev || !next) continue
    const oldName = str(prev.name).trim()
    const newName = str(next.name).trim()
    if (oldName && newName && oldName !== newName) {
      map.set(oldName, newName)
    }
  }
  return map
}

function matchNamesForCatalogEntry(
  nextName: string,
  renameMap: Map<string, string>,
): Set<string> {
  const names = new Set<string>()
  if (nextName) names.add(nextName)
  for (const [oldName, newName] of renameMap) {
    if (newName === nextName) names.add(oldName)
  }
  return names
}

function buildSyncOps(
  prevCatalog: Record<string, unknown>[],
  nextCatalog: Record<string, unknown>[],
  renameMap: Map<string, string>,
): SyncOp[] {
  const ops: SyncOp[] = []
  for (const catalog of nextCatalog) {
    const nextName = str(catalog.name).trim()
    if (!nextName) continue

    let prevRow = prevCatalog.find((p) => str(p.name).trim() === nextName)
    if (!prevRow) {
      for (const [oldName, newName] of renameMap) {
        if (newName === nextName) {
          prevRow = prevCatalog.find((p) => str(p.name).trim() === oldName)
          break
        }
      }
    }
    if (!prevRow) continue

    if (catalogCoreFingerprint(prevRow) === catalogCoreFingerprint(catalog)) {
      continue
    }

    ops.push({
      matchNames: matchNamesForCatalogEntry(nextName, renameMap),
      catalog,
      prevRow,
    })
  }
  return ops
}

function isServiceBackend(backend: Record<string, unknown>): boolean {
  const bt = str(backend.type)
  if (bt === 'handler' || bt === 'redirect') return false
  if (bt === 'service') return true
  return str(obj(backend.service).name).trim() !== ''
}

function findMatchingOp(
  backend: Record<string, unknown>,
  ops: SyncOp[],
): SyncOp | null {
  if (!isServiceBackend(backend)) return null
  const matchName = str(obj(backend.service).name).trim()
  if (!matchName) return null
  for (const op of ops) {
    if (op.matchNames.has(matchName)) return op
  }
  return null
}

function routeDriftFromPrevCatalog(
  backend: Record<string, unknown>,
  op: SyncOp,
): boolean {
  const svc = obj(backend.service)
  const routeFp = serviceCatalogCoreFingerprint(svc)
  const expectedFp = serviceCatalogCoreFingerprint(catalogRowToServiceCore(op.prevRow))
  return routeFp !== expectedFp
}

function conflictForBackend(
  backend: Record<string, unknown>,
  op: SyncOp,
  location: { id: string; label: string },
): ServiceRouteConflict | null {
  if (!routeDriftFromPrevCatalog(backend, op)) return null
  const svc = obj(backend.service)
  const matchName = str(svc.name).trim()
  return {
    id: location.id,
    label: location.label,
    serviceName: str(op.catalog.name).trim(),
    matchName,
    routeSummary: `${matchName}:${num(svc.port, 8080)} · ${str(svc.protocol, 'http') || 'http'}`,
  }
}

/** Routes whose catalog-core fields differ from the previous catalog snapshot. */
export function detectServiceRouteConflicts(
  rulesDoc: Record<string, unknown>,
  prevCatalog: Record<string, unknown>[],
  nextCatalog: Record<string, unknown>[],
): ServiceRouteConflict[] {
  const renameMap = buildServiceRenameMap(prevCatalog, nextCatalog)
  const ops = buildSyncOps(prevCatalog, nextCatalog, renameMap)
  if (ops.length === 0) return []

  const conflicts: ServiceRouteConflict[] = []
  const seen = new Set<string>()

  for (const [ri, rule] of rulesFromDoc(rulesDoc).entries()) {
    const host = str(rule.host) || `rules[${ri}]`
    const hostBackend = obj(rule.backend)
    const hostOp = findMatchingOp(hostBackend, ops)
    if (hostOp) {
      const id = `rule:${ri}:host`
      const c = conflictForBackend(hostBackend, hostOp, { id, label: `${host} · Host 级 backend` })
      if (c && !seen.has(id)) {
        seen.add(id)
        conflicts.push(c)
      }
    }

    for (const [pi, path] of arr<Record<string, unknown>>(rule.paths).entries()) {
      const pathBackend = obj(path.backend)
      const pathOp = findMatchingOp(pathBackend, ops)
      if (!pathOp) continue
      const pathLabel = str(path.path) || '/'
      const id = `rule:${ri}:path:${pi}`
      const c = conflictForBackend(pathBackend, pathOp, {
        id,
        label: `${host} · ${pathLabel}`,
      })
      if (c && !seen.has(id)) {
        seen.add(id)
        conflicts.push(c)
      }
    }
  }

  return conflicts
}

function resolvePatch(
  conflictId: string | null,
  conflicts: ServiceRouteConflict[],
  resolutions: ServiceRouteSyncResolutionMap,
): boolean {
  if (!conflictId) return true
  const isConflict = conflicts.some((c) => c.id === conflictId)
  if (!isConflict) return true
  return resolutions[conflictId] === 'overwrite'
}

function patchBackendIfAllowed(
  backend: Record<string, unknown>,
  ops: SyncOp[],
  conflictId: string | null,
  conflicts: ServiceRouteConflict[],
  resolutions: ServiceRouteSyncResolutionMap,
): boolean {
  if (!resolvePatch(conflictId, conflicts, resolutions)) return false
  const op = findMatchingOp(backend, ops)
  if (!op) return false
  const svc = obj(backend.service)
  backend.service = patchServiceObjectFromCatalog(svc, op.catalog)
  return true
}

/** Push catalog changes into matching route backends (respecting conflict resolutions). */
export function syncServicesCatalogToRules(
  rulesDoc: Record<string, unknown>,
  prevCatalog: Record<string, unknown>[],
  nextCatalog: Record<string, unknown>[],
  resolutions: ServiceRouteSyncResolutionMap = {},
): { rulesDoc: Record<string, unknown>; sync: ServiceRouteSyncResult } {
  const renameMap = buildServiceRenameMap(prevCatalog, nextCatalog)
  const ops = buildSyncOps(prevCatalog, nextCatalog, renameMap)
  const conflicts = detectServiceRouteConflicts(rulesDoc, prevCatalog, nextCatalog)

  if (ops.length === 0) {
    return { rulesDoc, sync: { touched: 0, skipped: 0, details: [] } }
  }

  let touched = 0
  let skipped = 0
  const details: string[] = []
  const rules = rulesFromDoc(rulesDoc)

  const nextRules = rules.map((rule, ri) => {
    const ruleCopy = { ...rule }
    const hostBackend = { ...obj(rule.backend) }
    const hostId = `rule:${ri}:host`
    if (patchBackendIfAllowed(hostBackend, ops, hostId, conflicts, resolutions)) {
      touched++
      details.push(`rules[${ri}] · ${str(rule.host) || 'host'}`)
    } else if (findMatchingOp(hostBackend, ops) && conflicts.some((c) => c.id === hostId)) {
      skipped++
    }
    ruleCopy.backend = hostBackend

    ruleCopy.paths = arr<Record<string, unknown>>(rule.paths).map((path, pi) => {
      const pathCopy = { ...path }
      const pathBackend = { ...obj(path.backend) }
      const pathId = `rule:${ri}:path:${pi}`
      if (patchBackendIfAllowed(pathBackend, ops, pathId, conflicts, resolutions)) {
        touched++
        const pathLabel = str(path.path) || '/'
        details.push(`rules[${ri}] · ${str(rule.host)} · ${pathLabel}`)
      } else if (findMatchingOp(pathBackend, ops) && conflicts.some((c) => c.id === pathId)) {
        skipped++
      }
      pathCopy.backend = pathBackend
      return pathCopy
    })

    return ruleCopy
  })

  return {
    rulesDoc: { ...rulesDoc, rules: nextRules },
    sync: { touched, skipped, details },
  }
}

export function defaultSyncResolutions(
  conflicts: ServiceRouteConflict[],
  choice: ServiceRouteSyncResolution,
): ServiceRouteSyncResolutionMap {
  const out: ServiceRouteSyncResolutionMap = {}
  for (const c of conflicts) {
    out[c.id] = choice
  }
  return out
}

export function mergeServicesDocsIntoRulesDoc(
  rulesDoc: Record<string, unknown>,
  servicesDoc: Record<string, unknown>,
  prevServicesDoc: Record<string, unknown>,
  resolutions: ServiceRouteSyncResolutionMap = {},
): { rulesDoc: Record<string, unknown>; sync: ServiceRouteSyncResult } {
  return syncServicesCatalogToRules(
    rulesDoc,
    servicesFromDoc(prevServicesDoc),
    servicesFromDoc(servicesDoc),
    resolutions,
  )
}

export function formatServiceRouteSyncMessage(sync: ServiceRouteSyncResult): string | null {
  const parts: string[] = []
  if (sync.touched > 0) parts.push(`已同步 ${sync.touched} 处路由 backend`)
  if (sync.skipped > 0) parts.push(`保留 ${sync.skipped} 处路由手动改动`)
  if (parts.length === 0) return null
  return parts.join('；')
}
