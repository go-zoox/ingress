import { describe, expect, it } from 'vitest'
import {
  emptyGlobalMaintenanceForm,
  validateGlobalMaintenanceForm,
  validateMaintenanceHostEntry,
  validateServiceMaintenanceForm,
  type GlobalMaintenanceForm,
} from './maintenance'

describe('validateMaintenanceHostEntry', () => {
  it('skips empty host rows', () => {
    expect(validateMaintenanceHostEntry({ host: '', window_start: '', window_end: '' }, 'hosts[1]')).toBeNull()
  })

  it('requires start and end', () => {
    expect(
      validateMaintenanceHostEntry(
        { host: 'app.example.com', window_start: '', window_end: '2026-06-01T00:00:00Z' },
        'hosts[1]',
      ),
    ).toMatch(/须填写维护开始与结束时间/)
  })

  it('rejects end before start', () => {
    expect(
      validateMaintenanceHostEntry(
        {
          host: 'app.example.com',
          window_start: '2026-06-02T00:00:00Z',
          window_end: '2026-06-01T00:00:00Z',
        },
        'hosts[1]',
      ),
    ).toMatch(/结束时间/)
  })
})

describe('validateGlobalMaintenanceForm', () => {
  it('allows empty hosts list', () => {
    expect(validateGlobalMaintenanceForm(emptyGlobalMaintenanceForm())).toBeNull()
  })

  it('requires window on each host', () => {
    const form: GlobalMaintenanceForm = {
      ...emptyGlobalMaintenanceForm(),
      maintenance_host_entries: [{ host: 'app.example.com', window_start: '', window_end: '' }],
    }
    expect(validateGlobalMaintenanceForm(form)).toMatch(/maintenance\.hosts\[1\]/)
  })
})

describe('validateServiceMaintenanceForm', () => {
  it('allows disabled maintenance', () => {
    expect(
      validateServiceMaintenanceForm({
        maintenance_enabled: false,
        maintenance_scope: 'all',
        maintenance_host_entries: [],
        maintenance_window_start: '',
        maintenance_window_end: '',
      }),
    ).toBeNull()
  })

  it('requires window for scope all', () => {
    expect(
      validateServiceMaintenanceForm({
        maintenance_enabled: true,
        maintenance_scope: 'all',
        maintenance_host_entries: [],
        maintenance_window_start: '',
        maintenance_window_end: '',
      }),
    ).toMatch(/service\.maintenance\.window/)
  })

  it('requires hosts for scope listed', () => {
    expect(
      validateServiceMaintenanceForm({
        maintenance_enabled: true,
        maintenance_scope: 'listed',
        maintenance_host_entries: [],
        maintenance_window_start: '',
        maintenance_window_end: '',
      }),
    ).toMatch(/至少添加一个 Host/)
  })
})
