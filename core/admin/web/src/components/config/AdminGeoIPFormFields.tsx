import { useEffect, useState } from 'react'
import { FormField } from '../Form'
import {
  adminGeoipFromDoc,
  adminGeoipToDoc,
  cloneAdminGeoIPForm,
  type AdminGeoIPForm,
} from '../../lib/adminGeoip'

export function AdminGeoIPFormFields({
  geoip,
  onChange,
}: {
  geoip: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
}) {
  const [form, setForm] = useState(() => adminGeoipFromDoc(geoip))

  useEffect(() => {
    setForm((prev) => {
      const built = adminGeoipToDoc(prev)
      const same =
        str(built.database) === str(geoip.database) &&
        str(built.ingress_label) === str(geoip.ingress_label) &&
        numOrEmpty(built.ingress_lat) === numOrEmpty(geoip.ingress_lat) &&
        numOrEmpty(built.ingress_lng) === numOrEmpty(geoip.ingress_lng)
      if (same) return prev
      return adminGeoipFromDoc(geoip)
    })
  }, [geoip])

  const patch = (fn: (next: AdminGeoIPForm) => void) => {
    setForm((prev) => {
      const next = cloneAdminGeoIPForm(prev)
      fn(next)
      onChange(adminGeoipToDoc(next))
      return next
    })
  }

  return (
    <>
      <FormField
        label="GeoLite2 数据库 admin.geoip.database"
        hint="MaxMind GeoLite2-City.mmdb 路径；默认 /etc/geoip/GeoLite2-City.mmdb。文件不存在或无读取权限时不启用 GeoIP，地图使用近似定位"
        value={form.database}
        onChange={(e) => patch((n) => { n.database = e.target.value })}
        placeholder="/etc/geoip/GeoLite2-City.mmdb"
      />
      <p className="form-hint">
        可从{' '}
        <a href="https://dev.maxmind.com/geoip/geolite2-free-geolocation-data" target="_blank" rel="noreferrer">
          MaxMind GeoLite2
        </a>{' '}
        下载；发布后热加载即可生效，无需重启 Admin。
      </p>
      <FormField
        label="Ingress 节点标签 admin.geoip.ingress_label"
        hint="WAF 攻击地图上本机（防御节点）显示名称"
        value={form.ingress_label}
        onChange={(e) => patch((n) => { n.ingress_label = e.target.value })}
        placeholder="上海"
      />
      <FormField
        label="Ingress 纬度 admin.geoip.ingress_lat"
        type="number"
        step="any"
        hint="留空时默认 31.2304（上海）"
        value={form.ingress_lat}
        onChange={(e) => patch((n) => { n.ingress_lat = e.target.value })}
        placeholder="31.2304"
      />
      <FormField
        label="Ingress 经度 admin.geoip.ingress_lng"
        type="number"
        step="any"
        hint="留空时默认 121.4737（上海）"
        value={form.ingress_lng}
        onChange={(e) => patch((n) => { n.ingress_lng = e.target.value })}
        placeholder="121.4737"
      />
    </>
  )
}

function str(v: unknown): string {
  return typeof v === 'string' ? v : v == null ? '' : String(v)
}

function numOrEmpty(v: unknown): string {
  if (typeof v === 'number' && Number.isFinite(v) && v !== 0) return String(v)
  return ''
}
