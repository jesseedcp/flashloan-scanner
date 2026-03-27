import type { ProtocolProgress } from '../types'
import { useI18n } from '../lib/i18n'

type ProtocolProgressCardProps = {
  progress: ProtocolProgress
}

export function ProtocolProgressCard({ progress }: ProtocolProgressCardProps) {
  const { t, protocolName, statusLabel } = useI18n()
  const totalBlocks = Math.max(progress.end_block - progress.start_block + 1, 1)
  const processedBlocks = Math.max(progress.current_block - progress.start_block + 1, 0)
  const ratio = Math.min(processedBlocks / totalBlocks, 1)

  return (
    <article className="protocol-card">
      <div className="protocol-card-top">
        <div>
          <p className="eyebrow">{protocolName(progress.protocol)}</p>
        </div>
        <div className={`status-badge status-${progress.status}`}>
          {statusLabel(progress.status)}
        </div>
      </div>

      <div className="protocol-stats-grid">
        <div className="protocol-stat">
          <span>{t.candidate}</span>
          <div className="metric-value protocol-metric-value">
            <strong>{progress.found_candidates}</strong>
            <span>{t.txUnit}</span>
          </div>
        </div>
        <div className="protocol-stat">
          <span>{t.verified}</span>
          <div className="metric-value protocol-metric-value">
            <strong>{progress.found_verified}</strong>
            <span>{t.txUnit}</span>
          </div>
        </div>
        <div className="protocol-stat">
          <span>{t.strict}</span>
          <div className="metric-value protocol-metric-value">
            <strong>{progress.found_strict}</strong>
            <span>{t.txUnit}</span>
          </div>
        </div>
      </div>

      <div className="progress-meta">
        <span className="progress-label">{t.blockProgress}</span>
        <span className="progress-value mono-text">
          {progress.current_block || progress.start_block} / {progress.end_block}
        </span>
      </div>
      <div className="progress-bar">
        <span style={{ width: `${ratio * 100}%` }} />
      </div>

      {progress.latest_finding ? (
        <div className="recent-finding">
          <span className="mono-text">{progress.latest_finding.tx_hash.slice(0, 12)}...</span>
          <strong className="finding-label">
            {progress.latest_finding.strict
              ? t.strict
              : progress.latest_finding.verified
                ? t.verified
                : t.candidate}
          </strong>
        </div>
      ) : (
        <p className="muted-text">{t.noFindingYet}</p>
      )}
      {progress.error ? <p className="error-text">{progress.error}</p> : null}
    </article>
  )
}
