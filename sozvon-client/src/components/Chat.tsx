// sozvon-client/src/components/Chat.tsx
import { useEffect, useState, useRef } from 'react'
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

  const messagesRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  // 🔹 Загрузка истории
  useEffect(() => {
    async function loadMessages() {
      const token = localStorage.getItem('token')

      const res = await fetch(
        `http://176.51.123.160:8080/chats/${chatId}/messages`,
        { headers: { Authorization: `Bearer ${token}` } }
      )

      if (!res.ok) return

      const data = await res.json()
      setMessages(Array.isArray(data) ? data : [])
    }

    setMessages([])
    loadMessages()
  }, [chatId])

  // 🔹 Фокус при открытии чата
  useEffect(() => {
    inputRef.current?.focus()
  }, [chatId])

  // 🔹 Автоскролл вниз
  useEffect(() => {
    if (messagesRef.current) {
      messagesRef.current.scrollTop = messagesRef.current.scrollHeight
    }
  }, [messages])

  // 🔹 WebSocket
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
      data: { chatId, text }
    })

    setText('')
  }

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
        maxWidth: 700,      // 🔥 ограничение ширины
        width: '100%'
      }}
    >
      <h3>Chat ID: {chatId}</h3>

      {/* Сообщения */}
      <div
        ref={messagesRef}
        style={{
          flex: 1,
          overflowY: 'auto',
          border: '1px solid #ddd',
          padding: 12,
          marginBottom: 12,
        }}
      >
        {messages.map(m => (
          <div key={m.id} style={{ marginBottom: 6 }}>
            <b>{m.from}:</b> {m.text}
          </div>
        ))}
      </div>

      {/* Ввод */}
      <div style={{ display: 'flex', gap: 10 }}>
        <input
          ref={inputRef}
          style={{
            flex: 1.5,        // 🔥 шире в 1.5 раза
            padding: 10,
            fontSize: 16
          }}
          value={text}
          maxLength={1000}
          onChange={e => setText(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && send()}
          placeholder="Type message..."
        />

        <button onClick={send}>Send</button>
      </div>
    </div>
  )
}
