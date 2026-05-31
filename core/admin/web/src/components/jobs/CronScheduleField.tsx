import {
  FormField,
  FormSelectField,
} from '../Form'
import { CRON_PRESET_CUSTOM, CRON_PRESETS, cronPresetForValue, isCustomCronPreset } from '../../lib/cronPresets'

type Props = {
  value: string
  onChange: (schedule: string) => void
}

export function CronScheduleField({ value, onChange }: Props) {
  const preset = cronPresetForValue(value)
  const custom = isCustomCronPreset(preset)
  const presetMeta = CRON_PRESETS.find((p) => p.value === value)

  return (
    <>
      <FormSelectField
        label="调度周期"
        hint={
          custom
            ? '五段式 Cron（分 时 日 月 周），或 @every 1s / @every 10s 等秒级周期'
            : presetMeta
              ? `${presetMeta.label} · ${value}`
              : value
                ? `当前表达式：${value}`
                : '选择常用周期，或切换为自定义'
        }
        full
        value={custom ? CRON_PRESET_CUSTOM : preset}
        onChange={(e) => {
          const next = e.target.value
          if (isCustomCronPreset(next)) {
            if (!custom) onChange('')
            return
          }
          onChange(next)
        }}
      >
        {CRON_PRESETS.map((p) => (
          <option key={p.id} value={p.value}>
            {p.label}
          </option>
        ))}
        <option value={CRON_PRESET_CUSTOM}>自定义 Cron 表达式</option>
      </FormSelectField>
      {custom ? (
        <FormField
          label="Cron 表达式"
          hint="例如 0 2 * * * 表示每天 02:00"
          full
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="0 2 * * *"
          spellCheck={false}
        />
      ) : null}
    </>
  )
}
