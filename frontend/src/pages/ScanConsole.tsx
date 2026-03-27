import { useEffect, useRef } from 'react'
import { FindingsTable } from '../components/FindingsTable'
import { LanguageToggle } from '../components/LanguageToggle'
import { LiveLogPanel } from '../components/LiveLogPanel'
import { ProtocolProgressCard } from '../components/ProtocolProgressCard'
import { ScanControlPanel } from '../components/ScanControlPanel'
import { useI18n } from '../lib/i18n'
import { createScanSocket, sendStartScan } from '../lib/ws'
import { useScanStore } from '../store/useScanStore'

export function ScanConsole() {
  const { t, statusLabel } = useI18n()
  const socketRef = useRef<WebSocket | null>(null)
  const socketStatus = useScanStore((state) => state.socketStatus)
  const params = useScanStore((state) => state.params)
  const summary = useScanStore((state) => state.summary)
  const protocols = useScanStore((state) => state.protocols)
  const findings = useScanStore((state) => state.findings)
  const logs = useScanStore((state) => state.logs)
  const jobStatus = useScanStore((state) => state.jobStatus)
  const error = useScanStore((state) => state.error)
  const setSocketStatus = useScanStore((state) => state.setSocketStatus)
  const startScan = useScanStore((state) => state.startScan)
  const applyMessage = useScanStore((state) => state.applyMessage)

  const isJobBusy = jobStatus === 'running' || jobStatus === 'pending'
  const isConnected = socketStatus === 'connected'
  const startDisabled = !isConnected || isJobBusy

  let startLabel = t.startScan
  let startHelperText = t.startButtonHintConnected
  if (!isConnected) {
    startLabel = t.waitingConnection
    startHelperText = t.startButtonHintDisconnected
  } else if (isJobBusy) {
    startLabel = t.scanInProgress
    startHelperText = t.startButtonHintRunning
  }

  useEffect(() => {
    setSocketStatus('connecting')
    const socket = createScanSocket(
      (message) => applyMessage(message),
      () => setSocketStatus('connected'),
      () => setSocketStatus('disconnected'),
    )
    socketRef.current = socket

    return () => {
      socket.close()
      socketRef.current = null
    }
  }, [applyMessage, setSocketStatus])

  return (
    <main className="page-shell">
      <section className="masthead panel">
        <div className="masthead-copy">
          <div className="hero-header">
            <div>
              <p className="eyebrow">{t.appEyebrow}</p>
              <h1>{t.heroTitle}</h1>
              <p className="hero-copy">{t.heroCopy}</p>
            </div>
            <LanguageToggle />
          </div>

          <div className="status-strip">
            <span>{t.socket}: {statusLabel(socketStatus)}</span>
            <span>{t.job}: {statusLabel(jobStatus)}</span>
            <span>{t.chain}: {params.chainId}</span>
            <span>{t.window}: {params.startBlock} - {params.endBlock}</span>
          </div>
        </div>

        <div className="signal-surface" aria-hidden="true">
          <div className="signal-grid" />
          <div className="signal-line signal-line-one" />
          <div className="signal-line signal-line-two" />
          <div className="signal-callout signal-callout-top">
            <span>{t.activeProtocols}</span>
            <strong>{params.protocols.length}</strong>
          </div>
          <div className="signal-callout signal-callout-bottom">
            <span>{t.totals}</span>
            <strong>{summary.totalStrict}{t.txUnit}</strong>
          </div>
        </div>
      </section>

      <ScanControlPanel
        defaultValues={params}
        disabled={startDisabled}
        startLabel={startLabel}
        helperText={startHelperText}
        onStart={(nextParams) => {
          startScan(nextParams)
          if (socketRef.current) {
            sendStartScan(socketRef.current, {
              chain_id: nextParams.chainId,
              start_block: nextParams.startBlock,
              end_block: nextParams.endBlock,
              trace_enabled: nextParams.traceEnabled,
              protocols: nextParams.protocols,
            })
          }
        }}
      />

      <section className="page-section">
        <section className="panel summary-panel">
          <div className="panel-header compact">
            <div>
              <p className="eyebrow">{t.jobOverview}</p>
              <h2>{t.totals}</h2>
              <p className="panel-copy">{t.jobOverviewCopy}</p>
            </div>
          </div>

          <div className="metric-grid">
            <div className="metric-cell">
              <span>{t.candidate}</span>
              <div className="metric-value">
                <strong>{summary.totalCandidates}</strong>
                <span>{t.txUnit}</span>
              </div>
            </div>
            <div className="metric-cell">
              <span>{t.verified}</span>
              <div className="metric-value">
                <strong>{summary.totalVerified}</strong>
                <span>{t.txUnit}</span>
              </div>
            </div>
            <div className="metric-cell">
              <span>{t.strict}</span>
              <div className="metric-value">
                <strong>{summary.totalStrict}</strong>
                <span>{t.txUnit}</span>
              </div>
            </div>
            <div className="metric-cell">
              <span>{t.protocolsCompleted}</span>
              <strong>{summary.completedProtocols}</strong>
            </div>
          </div>
          {error ? <p className="error-text inline-error">{error}</p> : null}
        </section>
      </section>

      <section className="page-section">
        <section className="panel protocol-runtime-panel">
          <div className="panel-header compact">
            <div>
              <p className="eyebrow">{t.protocol}</p>
              <h2>{t.protocolRuntime}</h2>
              <p className="panel-copy">{t.protocolRuntimeCopy}</p>
            </div>
          </div>
          <div className="protocol-runtime-grid">
            {Object.values(protocols).map((protocol) => (
              <ProtocolProgressCard key={protocol.protocol} progress={protocol} />
            ))}
          </div>
        </section>
      </section>

      <section className="data-grid">
        <FindingsTable items={findings} chainId={params.chainId} />
        <LiveLogPanel items={logs} />
      </section>
    </main>
  )
}
