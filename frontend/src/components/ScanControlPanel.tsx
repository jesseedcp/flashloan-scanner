import { useState } from 'react'

import { useI18n } from '../lib/i18n'

type ScanControlPanelProps = {
  defaultValues: {
    chainId: number
    startBlock: number
    endBlock: number
    traceEnabled: boolean
    protocols: string[]
  }
  disabled: boolean
  startLabel: string
  helperText: string
  onStart: (params: {
    chainId: number
    startBlock: number
    endBlock: number
    traceEnabled: boolean
    protocols: string[]
  }) => void
}

const protocolOptions = [
  { value: 'aave_v3', label: 'Aave V3' },
  { value: 'balancer_v2', label: 'Balancer V2' },
  { value: 'uniswap_v2', label: 'Uniswap V2' },
]

export function ScanControlPanel({ defaultValues, disabled, startLabel, helperText, onStart }: ScanControlPanelProps) {
  const { t, protocolName } = useI18n()
  const [chainId, setChainId] = useState(defaultValues.chainId)
  const [startBlock, setStartBlock] = useState(defaultValues.startBlock)
  const [endBlock, setEndBlock] = useState(defaultValues.endBlock)
  const [protocols, setProtocols] = useState(defaultValues.protocols)

  function toggleProtocol(protocol: string) {
    setProtocols((current) => {
      if (current.includes(protocol)) {
        return current.filter((item) => item !== protocol)
      }
      return [...current, protocol]
    })
  }

  const buttonDisabled = disabled || protocols.length === 0
  const resolvedHelperText = protocols.length === 0 ? t.startButtonHintSelectProtocol : helperText
  const resolvedButtonLabel = protocols.length === 0 ? t.selectOneProtocol : startLabel

  return (
    <section className="panel control-panel">
      <div className="panel-header">
        <div>
          <p className="eyebrow">{t.scanSetup}</p>
          <h2>{t.scanSetupTitle}</h2>
          <p className="panel-copy control-hint">{resolvedHelperText}</p>
        </div>
        <button
          className="primary-button"
          disabled={buttonDisabled}
          title={resolvedHelperText}
          onClick={() =>
            onStart({
              chainId,
              startBlock,
              endBlock,
              traceEnabled: defaultValues.traceEnabled,
              protocols,
            })
          }
        >
          {resolvedButtonLabel}
        </button>
      </div>

      <div className="form-grid">
        <label>
          <span>{t.chainId}</span>
          <input className="subtle-input" type="number" value={chainId} onChange={(event) => setChainId(Number(event.target.value))} />
        </label>
        <label>
          <span>{t.startBlock}</span>
          <input className="subtle-input" type="number" value={startBlock} onChange={(event) => setStartBlock(Number(event.target.value))} />
        </label>
        <label>
          <span>{t.endBlock}</span>
          <input className="subtle-input" type="number" value={endBlock} onChange={(event) => setEndBlock(Number(event.target.value))} />
        </label>
      </div>

      <div className="protocol-pills">
        {protocolOptions.map((protocol) => (
          <button
            key={protocol.value}
            className={`protocol-pill ${protocols.includes(protocol.value) ? 'selected' : ''}`}
            onClick={() => toggleProtocol(protocol.value)}
            type="button"
          >
            {protocolName(protocol.value)}
          </button>
        ))}
      </div>
    </section>
  )
}
