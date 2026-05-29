import { memo } from 'react'

export type HostTrafficRow = {
  name: string
  pv: number
  uv: number
}

type Props = {
  rows: HostTrafficRow[]
  maxRows?: number
}

export const HostTrafficTable = memo(function HostTrafficTable({ rows, maxRows }: Props) {
  if (rows.length === 0) {
    return <p className="empty-hint">无数据</p>
  }

  const visible = maxRows != null && maxRows > 0 ? rows.slice(0, maxRows) : rows
  const maxPV = Math.max(1, ...visible.map((r) => r.pv))

  return (
    <table className="data compact host-traffic-table">
      <thead>
        <tr>
          <th>域名</th>
          <th>PV</th>
          <th>UV</th>
          <th aria-hidden="true" className="host-traffic-bar-col" />
        </tr>
      </thead>
      <tbody>
        {visible.map((row) => (
          <tr key={row.name}>
            <td>
              <code className="host-traffic-name" title={row.name}>
                {row.name}
              </code>
            </td>
            <td>{row.pv.toLocaleString()}</td>
            <td>{row.uv.toLocaleString()}</td>
            <td className="host-traffic-bar-col">
              <div className="bar-track host-traffic-bar">
                <div className="bar-fill seg-2xx" style={{ width: `${(row.pv / maxPV) * 100}%` }} />
              </div>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
})
