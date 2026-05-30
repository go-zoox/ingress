import { arr, num, obj, str } from './ingressModuleForms'

export type MaintenanceScope = 'all' | 'listed'

export type MaintenanceHostFormEntry = {
  host: string
  window_start: string
  window_end: string
}

export type MaintenanceBypassFormSlice = {
  maintenance_bypass_paths: string
  maintenance_bypass_allow_ips: string
  maintenance_bypass_header_name: string
  maintenance_bypass_header_value: string
}

export type MaintenanceResponseHeaderFormSlice = {
  maintenance_response_header_name: string
  maintenance_response_header_value: string
}

export type MaintenanceStatusResponseFormSlice = {
  maintenance_status_response_ok: string
  maintenance_status_response_maintenance: string
  maintenance_status_response_content_type: string
}

export type GlobalMaintenanceForm = MaintenanceBypassFormSlice &
  MaintenanceResponseHeaderFormSlice &
  MaintenanceStatusResponseFormSlice & {
  maintenance_host_entries: MaintenanceHostFormEntry[]
  maintenance_retry_after: number
  maintenance_title: string
  maintenance_subtitle: string
  maintenance_status_path: string
}

export function emptyMaintenanceHostEntry(): MaintenanceHostFormEntry {
  return { host: '', window_start: '', window_end: '' }
}

export function emptyGlobalMaintenanceForm(): GlobalMaintenanceForm {
  return {
    maintenance_host_entries: [],
    maintenance_retry_after: 0,
    maintenance_title: '',
    maintenance_subtitle: '',
    maintenance_status_path: '',
    maintenance_bypass_paths: '',
    maintenance_bypass_allow_ips: '',
    maintenance_bypass_header_name: '',
    maintenance_bypass_header_value: '',
    maintenance_response_header_name: '',
    maintenance_response_header_value: '',
    maintenance_status_response_ok: '',
    maintenance_status_response_maintenance: '',
    maintenance_status_response_content_type: '',
  }
}

export function maintenanceHostEntriesFromYAML(raw: unknown): MaintenanceHostFormEntry[] {
  const items = arr<unknown>(raw)
  const out: MaintenanceHostFormEntry[] = []
  for (const item of items) {
    if (typeof item === 'string') {
      const host = item.trim()
      if (host) out.push({ host, window_start: '', window_end: '' })
      continue
    }
    const row = obj(item)
    const host = str(row.host).trim()
    if (!host) continue
    const window = obj(row.window)
    out.push({
      host,
      window_start: str(window.start),
      window_end: str(window.end),
    })
  }
  return out
}

export function maintenanceHostEntriesToYAML(entries: MaintenanceHostFormEntry[]): unknown[] {
  return entries
    .map((entry) => {
      const host = entry.host.trim()
      if (!host) return null
      const start = entry.window_start.trim()
      const end = entry.window_end.trim()
      if (!start && !end) return { host }
      const row: Record<string, unknown> = { host }
      const window: Record<string, unknown> = {}
      if (start) window.start = start
      if (end) window.end = end
      row.window = window
      return row
    })
    .filter(Boolean)
}

export function globalMaintenanceFromDoc(doc: Record<string, unknown>): GlobalMaintenanceForm {
  const m = obj(doc.maintenance)
  const bypass = obj(m.bypass)
  const header = obj(bypass.header)
  const responseHeader = obj(m.response_header)
  const statusResponse = obj(m.status_response)
  const paths = arr<string>(bypass.paths)
  const allowIPs = arr<string>(bypass.allow_ips)
  return {
    maintenance_host_entries: maintenanceHostEntriesFromYAML(m.hosts),
    maintenance_retry_after: num(m.retry_after, 0),
    maintenance_title: str(m.title),
    maintenance_subtitle: str(m.subtitle),
    maintenance_status_path: str(m.status_path),
    maintenance_bypass_paths: paths.join(', '),
    maintenance_bypass_allow_ips: allowIPs.join(', '),
    maintenance_bypass_header_name: str(header.name),
    maintenance_bypass_header_value: str(header.value),
    maintenance_response_header_name: str(responseHeader.name),
    maintenance_response_header_value: str(responseHeader.value),
    maintenance_status_response_ok: str(statusResponse.ok),
    maintenance_status_response_maintenance: str(statusResponse.maintenance),
    maintenance_status_response_content_type: str(statusResponse.content_type),
  }
}

function buildMaintenanceResponseHeader(
  form: MaintenanceResponseHeaderFormSlice,
): Record<string, unknown> | undefined {
  const name = form.maintenance_response_header_name.trim()
  const value = form.maintenance_response_header_value.trim()
  if (!name && !value) return undefined
  const block: Record<string, unknown> = {}
  if (name) block.name = name
  if (value) block.value = value
  return block
}

function buildMaintenanceStatusResponse(
  form: MaintenanceStatusResponseFormSlice,
): Record<string, unknown> | undefined {
  const ok = form.maintenance_status_response_ok.trim()
  const maintenance = form.maintenance_status_response_maintenance.trim()
  const contentType = form.maintenance_status_response_content_type.trim()
  if (!ok && !maintenance && !contentType) return undefined
  const block: Record<string, unknown> = {}
  if (ok) block.ok = ok
  if (maintenance) block.maintenance = maintenance
  if (contentType) block.content_type = contentType
  return block
}

function buildMaintenanceBypass(form: MaintenanceBypassFormSlice): Record<string, unknown> | undefined {
  const bypass: Record<string, unknown> = {}
  const paths = form.maintenance_bypass_paths.split(',').map((s) => s.trim()).filter(Boolean)
  if (paths.length) bypass.paths = paths
  const allowIPs = form.maintenance_bypass_allow_ips.split(',').map((s) => s.trim()).filter(Boolean)
  if (allowIPs.length) bypass.allow_ips = allowIPs
  if (form.maintenance_bypass_header_name.trim() || form.maintenance_bypass_header_value.trim()) {
    bypass.header = {
      name: form.maintenance_bypass_header_name.trim(),
      value: form.maintenance_bypass_header_value,
    }
  }
  return Object.keys(bypass).length ? bypass : undefined
}

export function globalMaintenanceConfigured(form: GlobalMaintenanceForm): boolean {
  return (
    form.maintenance_host_entries.length > 0 ||
    form.maintenance_retry_after > 0 ||
    form.maintenance_title.trim() !== '' ||
    form.maintenance_subtitle.trim() !== '' ||
    form.maintenance_status_path.trim() !== '' ||
    form.maintenance_bypass_paths.trim() !== '' ||
    form.maintenance_bypass_allow_ips.trim() !== '' ||
    form.maintenance_bypass_header_name.trim() !== '' ||
    form.maintenance_bypass_header_value.trim() !== '' ||
    form.maintenance_response_header_name.trim() !== '' ||
    form.maintenance_response_header_value.trim() !== '' ||
    form.maintenance_status_response_ok.trim() !== '' ||
    form.maintenance_status_response_maintenance.trim() !== '' ||
    form.maintenance_status_response_content_type.trim() !== ''
  )
}

export function patchGlobalMaintenance(
  doc: Record<string, unknown>,
  form: GlobalMaintenanceForm,
): Record<string, unknown> {
  if (!globalMaintenanceConfigured(form)) {
    const next = { ...doc }
    delete next.maintenance
    return next
  }
  const block: Record<string, unknown> = {}
  const hosts = maintenanceHostEntriesToYAML(form.maintenance_host_entries)
  if (hosts.length) block.hosts = hosts
  if (form.maintenance_retry_after > 0) block.retry_after = form.maintenance_retry_after
  if (form.maintenance_title.trim()) block.title = form.maintenance_title.trim()
  if (form.maintenance_subtitle.trim()) block.subtitle = form.maintenance_subtitle.trim()
  if (form.maintenance_status_path.trim()) block.status_path = form.maintenance_status_path.trim()
  const bypass = buildMaintenanceBypass(form)
  if (bypass) block.bypass = bypass
  const responseHeader = buildMaintenanceResponseHeader(form)
  if (responseHeader) block.response_header = responseHeader
  const statusResponse = buildMaintenanceStatusResponse(form)
  if (statusResponse) block.status_response = statusResponse
  return { ...doc, maintenance: block }
}

export function globalMaintenanceSectionOpen(form: GlobalMaintenanceForm): {
  hosts: boolean
  response: boolean
  statusApi: boolean
  bypass: boolean
} {
  const hasHosts = form.maintenance_host_entries.some((e) => e.host.trim() !== '')
  const hasResponse =
    form.maintenance_retry_after > 0 ||
    form.maintenance_title.trim() !== '' ||
    form.maintenance_subtitle.trim() !== '' ||
    form.maintenance_response_header_name.trim() !== '' ||
    form.maintenance_response_header_value.trim() !== ''
  const hasStatusApi =
    form.maintenance_status_path.trim() !== '' ||
    form.maintenance_status_response_ok.trim() !== '' ||
    form.maintenance_status_response_maintenance.trim() !== '' ||
    form.maintenance_status_response_content_type.trim() !== ''
  const hasBypass =
    form.maintenance_bypass_paths.trim() !== '' ||
    form.maintenance_bypass_allow_ips.trim() !== '' ||
    form.maintenance_bypass_header_name.trim() !== '' ||
    form.maintenance_bypass_header_value.trim() !== ''
  return {
    hosts: hasHosts,
    response: hasResponse,
    statusApi: hasStatusApi,
    bypass: hasBypass,
  }
}

export function maintenanceHostCount(form: GlobalMaintenanceForm): number {
  return form.maintenance_host_entries.filter((e) => e.host.trim()).length
}
