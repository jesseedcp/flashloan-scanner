import type { WSMessage } from '../types'

function getSocketUrl() {
  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
  return `${protocol}://${window.location.host}/ws/scan`
}

export function createScanSocket(onMessage: (message: WSMessage) => void, onOpen?: () => void, onClose?: () => void) {
  const socket = new WebSocket(getSocketUrl())

  socket.addEventListener('open', () => {
    onOpen?.()
  })

  socket.addEventListener('message', (event) => {
    const message = JSON.parse(event.data) as WSMessage
    onMessage(message)
  })

  socket.addEventListener('close', () => {
    onClose?.()
  })

  return socket
}

export function sendStartScan(
  socket: WebSocket,
  payload: {
    chain_id: number
    start_block: number
    end_block: number
    trace_enabled: boolean
    protocols: string[]
  },
) {
  socket.send(
    JSON.stringify({
      type: 'start_scan',
      payload,
    }),
  )
}
