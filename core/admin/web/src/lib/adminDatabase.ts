export type AdminDbDriver = 'sqlite' | 'mysql' | 'postgres'

export type SqliteDbForm = {
  path: string
  params: string
}

export type MysqlDbForm = {
  host: string
  port: number
  user: string
  password: string
  database: string
}

export type PostgresDbForm = {
  host: string
  port: number
  user: string
  password: string
  database: string
  sslmode: string
  timezone: string
}

export type AdminDatabaseForm = {
  driver: AdminDbDriver
  sqlite: SqliteDbForm
  mysql: MysqlDbForm
  postgres: PostgresDbForm
}

const DEFAULT_SQLITE: SqliteDbForm = {
  path: './admin.db',
  params: 'cache=shared&_fk=1',
}

const DEFAULT_MYSQL: MysqlDbForm = {
  host: '127.0.0.1',
  port: 3306,
  user: 'root',
  password: '',
  database: 'ingress',
}

const DEFAULT_POSTGRES: PostgresDbForm = {
  host: '127.0.0.1',
  port: 5432,
  user: 'postgres',
  password: '',
  database: 'ingress',
  sslmode: 'disable',
  timezone: 'UTC',
}

export function emptyAdminDatabaseForm(): AdminDatabaseForm {
  return {
    driver: 'sqlite',
    sqlite: { ...DEFAULT_SQLITE },
    mysql: { ...DEFAULT_MYSQL },
    postgres: { ...DEFAULT_POSTGRES },
  }
}

export function normalizeAdminDbDriver(raw: string): AdminDbDriver {
  switch (raw.trim().toLowerCase()) {
    case 'mysql':
      return 'mysql'
    case 'postgres':
    case 'postgresql':
      return 'postgres'
    default:
      return 'sqlite'
  }
}

export function adminDbDriverToYaml(driver: AdminDbDriver): string {
  return driver
}

export function looksLikeMysqlDsn(dsn: string): boolean {
  return dsn.includes('@tcp(')
}

export function looksLikePostgresDsn(dsn: string): boolean {
  const s = dsn.trim()
  return /\bhost=/.test(s) && /\b(dbname|port)=/.test(s)
}

export function looksLikeSqliteDsn(dsn: string): boolean {
  const s = dsn.trim()
  if (!s) return true
  if (s.startsWith('file:')) return true
  if (looksLikeMysqlDsn(s) || looksLikePostgresDsn(s)) return false
  return !s.includes('=') || s.endsWith('.db')
}

export function parseSqliteDsn(dsn: string): SqliteDbForm {
  const trimmed = dsn.trim()
  if (!trimmed || !looksLikeSqliteDsn(trimmed)) return { ...DEFAULT_SQLITE }
  if (trimmed.startsWith('file:')) {
    const rest = trimmed.slice('file:'.length)
    const q = rest.indexOf('?')
    if (q === -1) {
      return { path: rest || DEFAULT_SQLITE.path, params: DEFAULT_SQLITE.params }
    }
    return {
      path: rest.slice(0, q) || DEFAULT_SQLITE.path,
      params: rest.slice(q + 1) || DEFAULT_SQLITE.params,
    }
  }
  const q = trimmed.indexOf('?')
  if (q === -1) {
    return { path: trimmed, params: DEFAULT_SQLITE.params }
  }
  return {
    path: trimmed.slice(0, q) || DEFAULT_SQLITE.path,
    params: trimmed.slice(q + 1) || DEFAULT_SQLITE.params,
  }
}

export function parseMysqlDsn(dsn: string): MysqlDbForm {
  const trimmed = dsn.trim()
  if (!looksLikeMysqlDsn(trimmed)) return { ...DEFAULT_MYSQL }
  const m = trimmed.match(/^([^:@/]*):([^@]*)@tcp\(([^:]+):(\d+)\)\/([^?]+)/)
  if (!m) return { ...DEFAULT_MYSQL }
  const [, userEnc, passEnc, host, portStr, database] = m
  const decode = (s: string) => {
    try {
      return decodeURIComponent(s)
    } catch {
      return s
    }
  }
  return {
    host: host || DEFAULT_MYSQL.host,
    port: Number(portStr) || DEFAULT_MYSQL.port,
    user: decode(userEnc ?? '') || DEFAULT_MYSQL.user,
    password: decode(passEnc ?? ''),
    database: database || DEFAULT_MYSQL.database,
  }
}

function parsePostgresKV(dsn: string): Record<string, string> {
  const out: Record<string, string> = {}
  const re = /([A-Za-z_]+)=(?:"([^"]*)"|'([^']*)'|(\S+))/g
  let match: RegExpExecArray | null
  while ((match = re.exec(dsn)) !== null) {
    out[match[1].toLowerCase()] = match[2] ?? match[3] ?? match[4] ?? ''
  }
  return out
}

export function parsePostgresDsn(dsn: string): PostgresDbForm {
  const trimmed = dsn.trim()
  if (!looksLikePostgresDsn(trimmed)) return { ...DEFAULT_POSTGRES }
  const kv = parsePostgresKV(trimmed)
  if (Object.keys(kv).length === 0) return { ...DEFAULT_POSTGRES }
  return {
    host: kv.host || DEFAULT_POSTGRES.host,
    port: Number(kv.port) || DEFAULT_POSTGRES.port,
    user: kv.user || DEFAULT_POSTGRES.user,
    password: kv.password ?? DEFAULT_POSTGRES.password,
    database: kv.dbname || DEFAULT_POSTGRES.database,
    sslmode: kv.sslmode || DEFAULT_POSTGRES.sslmode,
    timezone: kv.timezone || DEFAULT_POSTGRES.timezone,
  }
}

export function cloneAdminDatabaseForm(form: AdminDatabaseForm): AdminDatabaseForm {
  return {
    driver: form.driver,
    sqlite: { ...form.sqlite },
    mysql: { ...form.mysql },
    postgres: { ...form.postgres },
  }
}

export function adminDatabaseFormFromConfig(driver: string, dsn: string): AdminDatabaseForm {
  const form = emptyAdminDatabaseForm()
  form.driver = normalizeAdminDbDriver(driver)
  switch (form.driver) {
    case 'mysql':
      form.mysql = parseMysqlDsn(dsn)
      break
    case 'postgres':
      form.postgres = parsePostgresDsn(dsn)
      break
    default:
      form.sqlite = parseSqliteDsn(dsn)
  }
  return form
}

export function buildSqliteDsn(sqlite: SqliteDbForm): string {
  const path = sqlite.path.trim() || DEFAULT_SQLITE.path
  const filePath = path.startsWith('file:') ? path : `file:${path}`
  const params = sqlite.params.trim()
  return params ? `${filePath}?${params}` : filePath
}

export function buildMysqlDsn(mysql: MysqlDbForm): string {
  const host = mysql.host.trim() || DEFAULT_MYSQL.host
  const port = mysql.port > 0 ? mysql.port : DEFAULT_MYSQL.port
  const user = encodeURIComponent(mysql.user.trim() || DEFAULT_MYSQL.user)
  const password = encodeURIComponent(mysql.password)
  const database = mysql.database.trim() || DEFAULT_MYSQL.database
  return `${user}:${password}@tcp(${host}:${port})/${database}?charset=utf8mb4&parseTime=True&loc=Local`
}

function postgresKV(key: string, value: string): string {
  const v = value.trim()
  if (v === '') return `${key}=`
  if (/[\s']/.test(v)) return `${key}='${v.replace(/'/g, "''")}'`
  return `${key}=${v}`
}

export function buildPostgresDsn(postgres: PostgresDbForm): string {
  const p = {
    host: postgres.host.trim() || DEFAULT_POSTGRES.host,
    port: postgres.port > 0 ? String(postgres.port) : String(DEFAULT_POSTGRES.port),
    user: postgres.user.trim() || DEFAULT_POSTGRES.user,
    password: postgres.password,
    database: postgres.database.trim() || DEFAULT_POSTGRES.database,
    sslmode: postgres.sslmode.trim() || DEFAULT_POSTGRES.sslmode,
    timezone: postgres.timezone.trim() || DEFAULT_POSTGRES.timezone,
  }
  return [
    postgresKV('host', p.host),
    postgresKV('user', p.user),
    postgresKV('password', p.password),
    postgresKV('dbname', p.database),
    postgresKV('port', p.port),
    postgresKV('sslmode', p.sslmode),
    postgresKV('TimeZone', p.timezone),
  ].join(' ')
}

export function buildAdminDatabaseDsn(form: AdminDatabaseForm): string {
  switch (form.driver) {
    case 'mysql':
      return buildMysqlDsn(form.mysql)
    case 'postgres':
      return buildPostgresDsn(form.postgres)
    default:
      return buildSqliteDsn(form.sqlite)
  }
}

export function adminDatabaseToConfig(form: AdminDatabaseForm): { driver: string; dsn: string } {
  return {
    driver: adminDbDriverToYaml(form.driver),
    dsn: buildAdminDatabaseDsn(form),
  }
}
