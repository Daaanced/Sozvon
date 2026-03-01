// sozvon-client/src/services/ws.ts

//import { parseToken } from '../functions/parse'

let socket: WebSocket | null = null
let listeners: ((msg: any) => void)[] = []
let isOpen = false

export function disconnectWS() {
  if (socket) {
    socket.close()
    socket = null
  }
  listeners = []
  isOpen = false
}

export function connectWS(token: string) {
  if (socket) return

  socket = new WebSocket(`ws://92.127.177.190:8080/ws?token=${token}`)

  socket.onopen = () => {
  console.log('[WS] connected')
  isOpen = true
}

socket.onclose = () => {
  isOpen = false
  socket = null
}


  socket.onmessage = (e) => {
    const msg = JSON.parse(e.data)
    listeners.forEach(fn => fn(msg))
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
