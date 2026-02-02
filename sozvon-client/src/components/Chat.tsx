//sozvon-client\src\components\Chat.tsx
import { useEffect, useState } from 'react'
import { connectChat, sendMessage, disconnectChat } from '../services/chat'

type Props = {
  chatId: string
  token: string
}

export default function Chat({ chatId, token }: Props) {
  const [messages, setMessages] = useState<any[]>([])
  const [text, setText] = useState('')

  useEffect(() => {
  connectChat(token, (msg) => {
    console.log("Received WS message:", msg)
    console.log("Current chatId:", chatId)

    // добавляем свои и чужие сообщения
    if (msg.event === 'message:new' && msg.data.chatId === chatId) {
      setMessages(prev => [...prev, {
        from: msg.data.from,
        text: msg.data.text
      }])
    }
  })

  return () => {
    disconnectChat()
  }
}, [token, chatId])

  return (
    <div>
      {messages.map((m, i) => (
        <div key={i}>
          <b>{m.from}:</b> {m.text}
        </div>
      ))}

      <input
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter') {
            sendMessage(chatId, text)
            setText('')
          }
        }}
      />
    </div>
  )
}

