// sozvon-client/src/services/ws.ts

let socket: WebSocket | null = null
let listeners: ((msg: any) => void)[] = []
let isOpen = false

export function connectWS(token: string) {
  if (socket) return

  socket = new WebSocket(`ws://90.189.252.24:8080/ws?token=${token}`)

  socket.onopen = () => {
    isOpen = true
    console.log('[WS] connected')
  }

  socket.onmessage = (e) => {
    console.log('[WS] raw message:', e.data)
    const msg = JSON.parse(e.data)
    listeners.forEach(fn => fn(msg))
  }

  socket.onclose = () => {
    socket = null
    isOpen = false
    listeners = []
  }
}

export function onWSMessage(fn: (msg: any) => void) {
  listeners.push(fn)
  return () => {
    listeners = listeners.filter(l => l !== fn)
  }
}

export function sendWS(data: any) {
  if (!socket || !isOpen) return
  socket.send(JSON.stringify(data))
}
