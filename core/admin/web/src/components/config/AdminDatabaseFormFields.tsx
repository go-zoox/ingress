import { useEffect, useState } from 'react'
import { FormField, FormSelectField } from '../Form'
import {
  adminDatabaseFormFromConfig,
  adminDatabaseToConfig,
  cloneAdminDatabaseForm,
  type AdminDatabaseForm,
  type AdminDbDriver,
} from '../../lib/adminDatabase'

const DRIVER_OPTIONS: { value: AdminDbDriver; label: string }[] = [
  { value: 'sqlite', label: 'SQLite' },
  { value: 'mysql', label: 'MySQL' },
  { value: 'postgres', label: 'PostgreSQL' },
]

export function AdminDatabaseFormFields({
  driver,
  dsn,
  onChange,
}: {
  driver: string
  dsn: string
  onChange: (next: { driver: string; dsn: string }) => void
}) {
  const [form, setForm] = useState(() => adminDatabaseFormFromConfig(driver, dsn))

  // Re-load from YAML only when props change externally (not from our own onChange).
  useEffect(() => {
    setForm((prev) => {
      const built = adminDatabaseToConfig(prev)
      if (built.driver === driver && built.dsn === dsn) {
        return prev
      }
      return adminDatabaseFormFromConfig(driver, dsn)
    })
  }, [driver, dsn])

  const patch = (fn: (next: AdminDatabaseForm) => void) => {
    setForm((prev) => {
      const next = cloneAdminDatabaseForm(prev)
      fn(next)
      onChange(adminDatabaseToConfig(next))
      return next
    })
  }

  return (
    <>
      <FormSelectField
        label="Driver"
        hint="Admin 审计 / 配置修订库；底层经 gormx 连接"
        value={form.driver}
        onChange={(e) => {
          patch((n) => {
            n.driver = e.target.value as AdminDbDriver
          })
        }}
      >
        {DRIVER_OPTIONS.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </FormSelectField>

      {form.driver === 'sqlite' && (
        <>
          <FormField
            label="数据库文件"
            hint="相对路径相对 ingress.yaml 所在目录；写入 DSN 时自动加 file: 前缀"
            value={form.sqlite.path}
            onChange={(e) => patch((n) => { n.sqlite.path = e.target.value })}
            placeholder="./admin.db"
          />
          <FormField
            label="连接参数"
            hint="SQLite query 字符串，如 cache=shared&_fk=1"
            value={form.sqlite.params}
            onChange={(e) => patch((n) => { n.sqlite.params = e.target.value })}
            placeholder="cache=shared&_fk=1"
          />
        </>
      )}

      {form.driver === 'mysql' && (
        <>
          <FormField
            label="Host"
            value={form.mysql.host}
            onChange={(e) => patch((n) => { n.mysql.host = e.target.value })}
            placeholder="127.0.0.1"
          />
          <FormField
            label="Port"
            type="number"
            value={form.mysql.port}
            onChange={(e) => patch((n) => { n.mysql.port = Number(e.target.value) })}
          />
          <FormField
            label="User"
            value={form.mysql.user}
            onChange={(e) => patch((n) => { n.mysql.user = e.target.value })}
          />
          <FormField
            label="Password"
            type="password"
            value={form.mysql.password}
            onChange={(e) => patch((n) => { n.mysql.password = e.target.value })}
            autoComplete="off"
          />
          <FormField
            label="Database"
            value={form.mysql.database}
            onChange={(e) => patch((n) => { n.mysql.database = e.target.value })}
          />
        </>
      )}

      {form.driver === 'postgres' && (
        <>
          <FormField
            label="Host"
            value={form.postgres.host}
            onChange={(e) => patch((n) => { n.postgres.host = e.target.value })}
            placeholder="127.0.0.1"
          />
          <FormField
            label="Port"
            type="number"
            value={form.postgres.port}
            onChange={(e) => patch((n) => { n.postgres.port = Number(e.target.value) })}
          />
          <FormField
            label="User"
            value={form.postgres.user}
            onChange={(e) => patch((n) => { n.postgres.user = e.target.value })}
          />
          <FormField
            label="Password"
            type="password"
            value={form.postgres.password}
            onChange={(e) => patch((n) => { n.postgres.password = e.target.value })}
            autoComplete="off"
          />
          <FormField
            label="Database"
            hint="对应 DSN 中的 dbname"
            value={form.postgres.database}
            onChange={(e) => patch((n) => { n.postgres.database = e.target.value })}
          />
          <FormField
            label="SSL mode"
            value={form.postgres.sslmode}
            onChange={(e) => patch((n) => { n.postgres.sslmode = e.target.value })}
            placeholder="disable"
          />
          <FormField
            label="TimeZone"
            value={form.postgres.timezone}
            onChange={(e) => patch((n) => { n.postgres.timezone = e.target.value })}
            placeholder="UTC"
          />
        </>
      )}
    </>
  )
}
