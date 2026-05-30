import { arr, num, obj, str, bool } from './ingressModuleForms'
import type { BackendForm } from './configEntities'

/** Catalog entry for reusable upstream services (ingress.yaml `services`). */
export type ServiceForm = {
  service_name: string
  service_port: number
  service_protocol: string
  service_mode: string
  health_check_enable: boolean
  health_check_method: string
  health_check_path: string
  health_check_status: string
  health_check_ok: boolean
  note: string
}

export function emptyServiceForm(): ServiceForm {
  return {
    service_name: '',
    service_port: 8080,
    service_protocol: 'http',
    service_mode: '',
    health_check_enable: false,
    health_check_method: 'GET',
    health_check_path: '/health',
    health_check_status: '',
    health_check_ok: false,
    note: '',
  }
}

export function servicesFromDoc(doc: Record<string, unknown>): Record<string, unknown>[] {
  return arr<Record<string, unknown>>(doc.services)
}

export function serviceToForm(row: Record<string, unknown>): ServiceForm {
  const hc = obj(row.healthcheck)
  const statusArr = arr<number>(hc.status)
  return {
    service_name: str(row.name),
    service_port: num(row.port, 8080),
    service_protocol: str(row.protocol, 'http') || 'http',
    service_mode: str(row.mode),
    health_check_enable: bool(hc.enable),
    health_check_method: str(hc.method, 'GET') || 'GET',
    health_check_path: str(hc.path, '/health') || '/health',
    health_check_status: statusArr.length > 0 ? statusArr.join(',') : '',
    health_check_ok: bool(hc.ok),
    note: str(row.note),
  }
}

function buildHealthCheck(form: ServiceForm): Record<string, unknown> | undefined {
  if (!form.health_check_enable) return undefined
  const hc: Record<string, unknown> = { enable: true }
  if (form.health_check_method && form.health_check_method !== 'GET') {
    hc.method = form.health_check_method
  }
  if (form.health_check_path && form.health_check_path !== '/health') {
    hc.path = form.health_check_path
  }
  if (form.health_check_status) {
    const codes = form.health_check_status
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)
      .map(Number)
      .filter((n) => !Number.isNaN(n))
    const nonDefault = codes.length > 0 && !(codes.length === 1 && codes[0] === 200)
    if (nonDefault) hc.status = codes
  }
  if (form.health_check_ok) hc.ok = true
  return hc
}

export function formToService(form: ServiceForm, original?: Record<string, unknown>): Record<string, unknown> {
  const row: Record<string, unknown> = { ...(original ?? {}) }
  row.name = form.service_name.trim()
  row.port = form.service_port
  if (form.service_protocol && form.service_protocol !== 'http') {
    row.protocol = form.service_protocol
  } else {
    delete row.protocol
  }
  if (form.service_mode === 'internal' || form.service_mode === 'external') {
    row.mode = form.service_mode
  } else {
    delete row.mode
  }
  const hcBlock = buildHealthCheck(form)
  if (hcBlock) row.healthcheck = hcBlock
  else delete row.healthcheck
  if (form.note.trim()) row.note = form.note.trim()
  else delete row.note
  return row
}

export function serviceSaveDisabled(form: ServiceForm): boolean {
  return !form.service_name.trim()
}

export function serviceSummary(row: Record<string, unknown>): string {
  const name = str(row.name)
  const port = num(row.port, 0)
  const protocol = str(row.protocol, 'http') || 'http'
  if (!name) return '—'
  return `${protocol}://${name}${port ? `:${port}` : ''}`
}

export function serviceFormListFromDoc(doc: Record<string, unknown>): ServiceForm[] {
  return servicesFromDoc(doc).map(serviceToForm)
}

export function applyServiceToBackend(form: ServiceForm, backend: BackendForm): BackendForm {
  return {
    ...backend,
    backend_type: 'service',
    service_name: form.service_name,
    service_port: form.service_port,
    service_protocol: form.service_protocol || 'http',
    service_mode: form.service_mode,
    health_check_enable: form.health_check_enable,
    health_check_method: form.health_check_method,
    health_check_path: form.health_check_path,
    health_check_status: form.health_check_status,
    health_check_ok: form.health_check_ok,
  }
}

export function serviceUsageCount(name: string, rulesDoc: Record<string, unknown>): number {
  const target = name.trim()
  if (!target) return 0
  let count = 0
  for (const rule of arr<Record<string, unknown>>(rulesDoc.rules)) {
    const backend = obj(rule.backend)
    const svc = obj(backend.service)
    if (str(svc.name) === target) count++
    for (const path of arr<Record<string, unknown>>(rule.paths)) {
      const pb = obj(path.backend)
      const ps = obj(pb.service)
      if (str(ps.name) === target) count++
    }
  }
  return count
}
