// src/services/chat.ts
let socket: WebSocket | null = null

export function connectChat(
  token: string,
  onMessage: (msg: any) => void
) {
  if (socket) {
    socket.close()
  }

  socket = new WebSocket(`ws://90.189.252.24:8080/ws?token=${token}`)

  socket.onmessage = (event) => {
    onMessage(JSON.parse(event.data))
  }
}

export function sendMessage(chatId: string, text: string) {
  if (!socket) return

  socket.send(
    JSON.stringify({
      event: 'message:send',
      data: {
        chatId,
        text
      }
    })
  )
}

export function disconnectChat() {
  if (socket) {
    socket.close()
    socket = null
  }
}