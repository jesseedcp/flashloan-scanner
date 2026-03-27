import { Link } from 'react-router-dom'
import type { FindingItem } from '../types'
import { useI18n } from '../lib/i18n'

type FindingsTableProps = {
  items: FindingItem[]
  chainId: number
}

export function FindingsTable({ items, chainId }: FindingsTableProps) {
  const { t, protocolName, boolLabel } = useI18n()
  return (
    <section className="panel table-panel">
      <div className="panel-header compact">
        <div>
          <p className="eyebrow">{t.liveFindings}</p>
          <h2>{t.newestOnTop}</h2>
        </div>
      </div>

      <div className="table-shell">
        <table>
          <thead>
            <tr>
              <th>{t.txHash}</th>
              <th>{t.protocol}</th>
              <th>{t.block}</th>
              <th>{t.candidate}</th>
              <th>{t.verified}</th>
              <th>{t.strict}</th>
            </tr>
          </thead>
          <tbody>
            {items.length === 0 ? (
              <tr>
                <td colSpan={6} className="empty-cell">
                  {t.findingsWaiting}
                </td>
              </tr>
            ) : (
              items.map((item) => (
                <tr key={`${item.protocol}-${item.tx_hash}`}>
                  <td className="tx-hash-cell">
                    <span className="hash-tooltip-wrap">
                      <Link className="mono-link truncated-hash-link" to={`/tx/${item.tx_hash}?chain_id=${chainId}`}>
                        {shortenHash(item.tx_hash)}
                      </Link>
                      <span className="hash-tooltip" role="tooltip">
                        {item.tx_hash}
                      </span>
                    </span>
                  </td>
                  <td>{protocolName(item.protocol)}</td>
                  <td>{item.block_number}</td>
                  <td><span className={`table-badge ${item.candidate ? 'positive' : 'muted'}`}>{boolLabel(item.candidate)}</span></td>
                  <td><span className={`table-badge ${item.verified ? 'positive' : 'muted'}`}>{boolLabel(item.verified)}</span></td>
                  <td><span className={`table-badge ${item.strict ? 'positive' : 'muted'}`}>{boolLabel(item.strict)}</span></td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </section>
  )
}

function shortenHash(value: string) {
  if (value.length <= 18) {
    return value
  }
  return `${value.slice(0, 10)}...${value.slice(-6)}`
}
