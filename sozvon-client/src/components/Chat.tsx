// sozvon-client/src/components/Chat.tsx
import { useEffect, useState, useRef } from 'react'
import { onWSMessage, sendWS } from '../services/ws'
import { v4 as uuidv4 } from "uuid"
import { useChatContext } from '../context/ChatContext'

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
  const { getSafeUser } = useChatContext()
  const [messages, setMessages] = useState<Message[]>([])
  const [text, setText] = useState('')

  const messagesRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  // 🔹 Загрузка истории
  useEffect(() => {
    async function loadMessages() {
      const token = localStorage.getItem('token')

      const res = await fetch(
        `http://92.127.177.190:8080/chats/${chatId}/messages`,
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

function formatTime(dateString: string) {
  const d = new Date(dateString)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function formatFullDate(dateString: string) {
  const d = new Date(dateString)
  return d.toLocaleDateString('ru-RU') + ', ' +
         d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function isSameDay(a: string, b: string) {
  const d1 = new Date(a)
  const d2 = new Date(b)

  return (
    d1.getFullYear() === d2.getFullYear() &&
    d1.getMonth() === d2.getMonth() &&
    d1.getDate() === d2.getDate()
  )
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
{messages.map((m, index) => {
  const user = getSafeUser(m.from)
  const prev = messages[index - 1]

  const isGroupStart =
    !prev ||
    prev.from !== m.from ||
    !isSameDay(prev.createdAt, m.createdAt)

  if (isGroupStart) {
    // 🟢 Тип 1 — с аватаром
    return (
      <div key={m.id} style={styles.groupStartWrapper}>
        <img
          src={user.picture}
          style={styles.avatar}
        />

        <div style={styles.groupContent}>
          <div style={styles.headerLine}>
            <span style={styles.userName}>
              {user.name}
            </span>

            <span style={styles.fullDate}>
              {formatFullDate(m.createdAt)}
            </span>
          </div>

          <div style={styles.messageText}>
            {m.text}
          </div>
        </div>
      </div>
    )
  }

  // 🔵 Тип 2 — внутри группы
  return (
  <div key={m.id} style={styles.groupStartWrapper}>
    {/* ЛЕВАЯ КОЛОНКА (под аватаром) */}
    <div style={styles.timeColumn}>
      <span style={styles.smallTime}>
        {formatTime(m.createdAt)}
      </span>
    </div>

    {/* ПРАВАЯ КОЛОНКА */}
    <div style={styles.groupContent}>
      <div style={styles.messageText}>
        {m.text}
      </div>
    </div>
  </div>
)
})}
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

const styles = {
  groupStartWrapper: {
    display: 'flex',
    gap: 12,
    marginTop: 2,
  },

  avatar: {
    width: 40,
    height: 40,
    borderRadius: '50%',
    objectFit: 'cover' as const,
    flexShrink: 0,
  },

  groupContent: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column' as const,
  },

  headerLine: {
    display: 'flex',
    alignItems: 'baseline',
    gap: 8,
  },

  userName: {
    fontWeight: 600,
    fontSize: 15,
  },

  fullDate: {
    fontSize: 12,
    color: '#888',
  },

  messageText: {
    marginTop: 4,
    lineHeight: 1.4,
    wordBreak: 'break-word' as const,
  },

  timeColumn: {
	width: 40,
	display: 'flex',
	justifyContent: 'center',
	alignItems: 'baseline',
	paddingTop: 4, // чуть ниже верхней линии
   },

  smallTime: {
	fontSize: 12,
	color: '#888',
	width: 40,
	flexShrink: 0,
	marginTop: 4,
	textAlign: 'center' as const,     // ← вот это главное
	},
}