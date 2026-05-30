import { num, setStr, str } from './ingressModuleForms'

export type AdminGeoIPForm = {
  database: string
  ingress_label: string
  ingress_lat: string
  ingress_lng: string
}

export function emptyAdminGeoIPForm(): AdminGeoIPForm {
  return {
    database: '',
    ingress_label: '',
    ingress_lat: '',
    ingress_lng: '',
  }
}

export function cloneAdminGeoIPForm(form: AdminGeoIPForm): AdminGeoIPForm {
  return { ...form }
}

export function adminGeoipFromDoc(geoip: Record<string, unknown>): AdminGeoIPForm {
  const lat = num(geoip.ingress_lat, 0)
  const lng = num(geoip.ingress_lng, 0)
  return {
    database: str(geoip.database),
    ingress_label: str(geoip.ingress_label),
    ingress_lat: lat !== 0 ? String(lat) : '',
    ingress_lng: lng !== 0 ? String(lng) : '',
  }
}

export function adminGeoipToDoc(form: AdminGeoIPForm): Record<string, unknown> {
  const doc: Record<string, unknown> = {}
  setStr(doc, 'database', form.database)
  setStr(doc, 'ingress_label', form.ingress_label)
  const lat = form.ingress_lat.trim()
  const lng = form.ingress_lng.trim()
  if (lat !== '') {
    const n = Number(lat)
    if (Number.isFinite(n)) doc.ingress_lat = n
  }
  if (lng !== '') {
    const n = Number(lng)
    if (Number.isFinite(n)) doc.ingress_lng = n
  }
  return doc
}
