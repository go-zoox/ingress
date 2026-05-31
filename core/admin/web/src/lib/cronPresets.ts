/** Common cron presets (5-field or @every for sub-minute). */
export const CRON_PRESET_CUSTOM = '__custom__'

export type CronPreset = {
  id: string
  label: string
  value: string
}

export const CRON_PRESETS: CronPreset[] = [
  { id: 'every-1s', label: '每秒', value: '@every 1s' },
  { id: 'every-10s', label: '每 10 秒', value: '@every 10s' },
  { id: 'every-30s', label: '每 30 秒', value: '@every 30s' },
  { id: 'every-1m', label: '每分钟', value: '*/1 * * * *' },
  { id: 'every-30m', label: '每 30 分钟', value: '*/30 * * * *' },
  { id: 'hourly', label: '每小时整点', value: '0 * * * *' },
  { id: 'every-6h', label: '每 6 小时', value: '0 */6 * * *' },
  { id: 'daily-2', label: '每天 02:00', value: '0 2 * * *' },
  { id: 'daily-3', label: '每天 03:00', value: '0 3 * * *' },
  { id: 'daily-4', label: '每天 04:00', value: '0 4 * * *' },
  { id: 'weekly-sun', label: '每周日 04:00', value: '0 4 * * 0' },
  { id: 'monthly-1', label: '每月 1 日 00:00', value: '0 0 1 * *' },
]

export function cronPresetForValue(schedule: string): string {
  const trimmed = schedule.trim()
  const hit = CRON_PRESETS.find((p) => p.value === trimmed)
  return hit ? hit.value : CRON_PRESET_CUSTOM
}

export function isCustomCronPreset(presetValue: string): boolean {
  return presetValue === CRON_PRESET_CUSTOM
}
