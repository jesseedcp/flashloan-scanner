import type { ReactNode } from 'react'
import type { LogItem } from '../types'
import { useI18n } from '../lib/i18n'

type LiveLogPanelProps = {
  items: LogItem[]
}

export function LiveLogPanel({ items }: LiveLogPanelProps) {
  const { t, protocolName, logMessage } = useI18n()
  return (
    <section className="panel log-panel">
      <div className="panel-header compact">
        <div>
          <p className="eyebrow">{t.liveLog}</p>
          <h2>{t.runtimeSignals}</h2>
        </div>
      </div>
      <div className="log-list">
        {items.length === 0 ? <p className="muted-text">{t.waitingForEvents}</p> : null}
        {items.map((item, index) => (
          <div key={`${item.timestamp}-${index}`} className={`log-item ${item.level}`}>
            <span className="log-time">{new Date(item.timestamp).toLocaleTimeString()}</span>
            <div className="log-body">
              <p>
                {item.protocol ? `[${protocolName(item.protocol)}] ` : ''}
                {renderLogMessage(logMessage(item.message, item.protocol))}
              </p>
              {item.block_number ? <span className="log-block">{t.block} {item.block_number}</span> : null}
            </div>
          </div>
        ))}
      </div>
    </section>
  )
}

function renderLogMessage(message: string) {
  const hashPattern = /(0x[a-fA-F0-9]{40,64})/g
  const matches = Array.from(message.matchAll(hashPattern))
  if (matches.length === 0) {
    return message
  }

  const nodes: ReactNode[] = []
  let lastIndex = 0

  matches.forEach((match, index) => {
    const fullHash = match[0]
    const startIndex = match.index ?? 0
    if (startIndex > lastIndex) {
      nodes.push(message.slice(lastIndex, startIndex))
    }
    nodes.push(
      <span key={`${fullHash}-${index}`} className="log-hash-wrap">
        <span className="log-hash" tabIndex={0}>
          {shortenHash(fullHash)}
        </span>
        <span className="log-hash-tooltip" role="tooltip">
          {fullHash}
        </span>
      </span>,
    )
    lastIndex = startIndex + fullHash.length
  })

  if (lastIndex < message.length) {
    nodes.push(message.slice(lastIndex))
  }

  return nodes
}

function shortenHash(value: string) {
  if (value.length <= 18) {
    return value
  }
  return `${value.slice(0, 10)}...${value.slice(-6)}`
}
