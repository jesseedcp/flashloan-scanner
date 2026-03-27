import { create } from 'zustand'
import type { FindingItem, JobOverview, JobStatus, LogItem, ProtocolProgress, SocketStatus, WSMessage } from '../types'

type ScanParams = {
  chainId: number
  startBlock: number
  endBlock: number
  traceEnabled: boolean
  protocols: string[]
}

type ScanState = {
  socketStatus: SocketStatus
  jobStatus: JobStatus
  jobId?: string
  params: ScanParams
  summary: {
    totalCandidates: number
    totalVerified: number
    totalStrict: number
    completedProtocols: number
  }
  protocols: Record<string, ProtocolProgress>
  findings: FindingItem[]
  logs: LogItem[]
  error?: string
  setSocketStatus: (status: SocketStatus) => void
  startScan: (params: ScanParams) => void
  applyMessage: (message: WSMessage) => void
}

const defaultProtocols = ['aave_v3', 'balancer_v2', 'uniswap_v2']

function extractTxHash(message: string) {
  const match = message.match(/0x[a-fA-F0-9]{64}/)
  return match?.[0]
}

function enrichLogWithFinding(log: LogItem, findings: FindingItem[]) {
  if (log.block_number) {
    return log
  }
  const txHash = extractTxHash(log.message)
  if (!txHash) {
    return log
  }
  const finding = findings.find((item) => item.tx_hash.toLowerCase() === txHash.toLowerCase())
  if (!finding) {
    return log
  }
  return {
    ...log,
    block_number: finding.block_number,
  }
}

function buildInitialProtocolMap() {
  return Object.fromEntries(
    defaultProtocols.map((protocol) => [
      protocol,
      {
        protocol,
        status: 'idle' as JobStatus,
        start_block: 22485844,
        end_block: 22486844,
        current_block: 0,
        found_candidates: 0,
        found_verified: 0,
        found_strict: 0,
      },
    ]),
  ) as Record<string, ProtocolProgress>
}

function applyOverview(state: ScanState, overview: JobOverview) {
  return {
    ...state,
    jobId: overview.job_id,
    jobStatus: overview.status,
    error: overview.error,
    summary: {
      totalCandidates: overview.total_candidates,
      totalVerified: overview.total_verified,
      totalStrict: overview.total_strict,
      completedProtocols: overview.completed_protocols,
    },
    protocols: {
      ...state.protocols,
      ...overview.protocols,
    },
  }
}

export const useScanStore = create<ScanState>((set) => ({
  socketStatus: 'disconnected',
  jobStatus: 'idle',
  params: {
    chainId: 1,
    startBlock: 22485844,
    endBlock: 22486844,
    traceEnabled: true,
    protocols: defaultProtocols,
  },
  summary: {
    totalCandidates: 0,
    totalVerified: 0,
    totalStrict: 0,
    completedProtocols: 0,
  },
  protocols: buildInitialProtocolMap(),
  findings: [],
  logs: [],
  setSocketStatus: (status) => set({ socketStatus: status }),
  startScan: (params) =>
    set({
      params,
      jobStatus: 'pending',
      jobId: undefined,
      findings: [],
      logs: [],
      error: undefined,
      summary: {
        totalCandidates: 0,
        totalVerified: 0,
        totalStrict: 0,
        completedProtocols: 0,
      },
      protocols: Object.fromEntries(
        params.protocols.map((protocol) => [
          protocol,
          {
            protocol,
            status: 'pending' as JobStatus,
            start_block: params.startBlock,
            end_block: params.endBlock,
            current_block: 0,
            found_candidates: 0,
            found_verified: 0,
            found_strict: 0,
          },
        ]),
      ),
    }),
  applyMessage: (message) =>
    set((state) => {
      switch (message.type) {
        case 'job_started':
        case 'job_progress':
        case 'job_completed':
        case 'job_failed':
          return applyOverview(state, message.payload as JobOverview)
        case 'protocol_progress':
        case 'protocol_completed':
        case 'protocol_failed': {
          const payload = message.payload as ProtocolProgress
          return {
            ...state,
            protocols: {
              ...state.protocols,
              [payload.protocol]: payload,
            },
          }
        }
        case 'finding': {
          const finding = message.payload as FindingItem
          return {
            ...state,
            findings: [finding, ...state.findings],
          }
        }
        case 'log': {
          const log = enrichLogWithFinding(message.payload as LogItem, state.findings)
          return {
            ...state,
            logs: [log, ...state.logs].slice(0, 200),
          }
        }
        case 'error':
          return {
            ...state,
            jobStatus: 'failed',
            error: (message.payload as { message?: string }).message ?? 'Unknown websocket error',
          }
        default:
          return state
      }
    }),
}))
