// sozvon-client/src/components/Chat.tsx

import { useEffect, useState } from 'react'
import { onWSMessage, sendWS } from '../services/ws'
import { v4 as uuidv4 } from "uuid"

type Props = {
  chatId: string
}

type Message = {
  id: string
  from: string
  text: string
  createdAt: string
}

export default function Chat({ chatId }: Props) {
  const [messages, setMessages] = useState<Message[]>([])
  const [text, setText] = useState('')

  // ✅ Загружаем историю при смене chatId
  useEffect(() => {
    async function loadMessages() {
      const token = localStorage.getItem('token')

      const res = await fetch(
        `http://176.51.121.88:8080/chats/${chatId}/messages`,
        {
          headers: {
            Authorization: `Bearer ${token}`
          }
        }
      )

      if (!res.ok) {
        console.error('failed to load messages')
        return
      }

      const data = await res.json()
      setMessages(Array.isArray(data) ? data : [])
    }

    setMessages([]) // очистка при смене чата
    loadMessages()
  }, [chatId])

  // ✅ Слушаем новые сообщения
  useEffect(() => {
    const off = onWSMessage(msg => {
      if (msg.event === 'message:new' && msg.data.chatId === chatId) {
        setMessages(prev => [
          ...prev,
          {
            id: uuidv4(),
            from: msg.data.from,
            text: msg.data.text,
            createdAt: new Date().toISOString()
          }
        ])
      }
    })

    return off
  }, [chatId])

  function send() {
    if (!text.trim()) return

    sendWS({
      event: 'message:send',
      data: {
        chatId,
        text
      }
    })

    setText('')
  }

  return (
    <div>
      <h3>Chat ID: {chatId}</h3>

      <div style={{ marginBottom: 20 }}>
        {messages.map(m => (
          <div key={m.id}>
            <b>{m.from}:</b> {m.text}
          </div>
        ))}
      </div>

      <input
        value={text}
        onChange={e => setText(e.target.value)}
        onKeyDown={e => {
          if (e.key === 'Enter') send()
        }}
        placeholder="Type message..."
      />
    </div>
  )
}
