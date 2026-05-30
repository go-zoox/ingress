import { describe, expect, it } from 'vitest'
import {
  buildServiceRenameMap,
  detectServiceRouteConflicts,
  patchServiceObjectFromCatalog,
  syncServicesCatalogToRules,
} from './serviceRouteSync'

const prevCatalog = [
  { name: 'api.internal', port: 8080, protocol: 'http' },
]

const nextCatalogPort = [
  { name: 'api.internal', port: 9090, protocol: 'http' },
]

function rulesDocWithBackend(name: string, port: number, host = 'api.example.com') {
  return {
    rules: [
      {
        host,
        backend: {
          type: 'service',
          service: { name, port, protocol: 'http' },
        },
      },
    ],
  }
}

describe('syncServicesCatalogToRules', () => {
  it('updates in-sync route backends when catalog port changes', () => {
    const rulesDoc = rulesDocWithBackend('api.internal', 8080)
    const { rulesDoc: out, sync } = syncServicesCatalogToRules(rulesDoc, prevCatalog, nextCatalogPort)
    expect(sync.touched).toBe(1)
    const svc = (out.rules as Record<string, unknown>[])[0].backend as Record<string, unknown>
    expect((svc.service as Record<string, unknown>).port).toBe(9090)
  })

  it('detects manual drift when route port differs from prev catalog', () => {
    const rulesDoc = rulesDocWithBackend('api.internal', 3000)
    const conflicts = detectServiceRouteConflicts(rulesDoc, prevCatalog, nextCatalogPort)
    expect(conflicts).toHaveLength(1)
    expect(conflicts[0].id).toBe('rule:0:host')
  })

  it('keeps manual route backend when resolution is keep', () => {
    const rulesDoc = rulesDocWithBackend('api.internal', 3000)
    const { rulesDoc: out, sync } = syncServicesCatalogToRules(rulesDoc, prevCatalog, nextCatalogPort, {
      'rule:0:host': 'keep',
    })
    expect(sync.skipped).toBe(1)
    expect(sync.touched).toBe(0)
    const svc = (out.rules as Record<string, unknown>[])[0].backend as Record<string, unknown>
    expect((svc.service as Record<string, unknown>).port).toBe(3000)
  })

  it('overwrites manual route backend when resolution is overwrite', () => {
    const rulesDoc = rulesDocWithBackend('api.internal', 3000)
    const { rulesDoc: out, sync } = syncServicesCatalogToRules(rulesDoc, prevCatalog, nextCatalogPort, {
      'rule:0:host': 'overwrite',
    })
    expect(sync.touched).toBe(1)
    const svc = (out.rules as Record<string, unknown>[])[0].backend as Record<string, unknown>
    expect((svc.service as Record<string, unknown>).port).toBe(9090)
  })

  it('follows rename map for route refs with old service name', () => {
    const prev = [{ name: 'api.old', port: 8080 }]
    const next = [{ name: 'api.new', port: 8080 }]
    const rulesDoc = rulesDocWithBackend('api.old', 8080)
    const rename = buildServiceRenameMap(prev, next)
    expect(rename.get('api.old')).toBe('api.new')

    const { rulesDoc: out, sync } = syncServicesCatalogToRules(rulesDoc, prev, next)
    expect(sync.touched).toBe(1)
    const svc = (out.rules as Record<string, unknown>[])[0].backend as Record<string, unknown>
    expect((svc.service as Record<string, unknown>).name).toBe('api.new')
  })

  it('preserves route-specific auth when syncing catalog fields', () => {
    const rulesDoc = {
      rules: [
        {
          host: 'api.example.com',
          backend: {
            type: 'service',
            service: {
              name: 'api.internal',
              port: 8080,
              auth: { type: 'basic', basic: { users: [{ username: 'u', password: 'p' }] } },
            },
          },
        },
      ],
    }
    const { rulesDoc: out } = syncServicesCatalogToRules(rulesDoc, prevCatalog, nextCatalogPort)
    const svc = ((out.rules as Record<string, unknown>[])[0].backend as Record<string, unknown>)
      .service as Record<string, unknown>
    expect(svc.port).toBe(9090)
    expect(svc.auth).toBeTruthy()
  })
})

describe('patchServiceObjectFromCatalog', () => {
  it('maps healthcheck from catalog row', () => {
    const row = {
      name: 'x',
      port: 8080,
      healthcheck: { enable: true, path: '/ready' },
    }
    const patched = patchServiceObjectFromCatalog({ name: 'x', port: 8080 }, row)
    expect(patched.healthcheck).toEqual({ enable: true, path: '/ready' })
  })
})
