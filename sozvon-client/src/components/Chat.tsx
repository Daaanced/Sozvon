// sozvon-client/src/components/Chat.tsx
import { useEffect, useState, useRef } from 'react'
import { onWSMessage } from '../services/ws'
import { getMessages, sendMessage } from '../api/chats'
import { useChatContext } from '../context/ChatContext'

type Props = { chatId: string }

type Message = {
  id: string
  senderId: number
  text: string
  createdAt: string
}

export default function Chat({ chatId }: Props) {
  const { getSafeUser, markRead, notifyOwnMessage } = useChatContext()
  const [messages, setMessages] = useState<Message[]>([])
  const [text, setText] = useState('')
  const messagesRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  // При открытии чата — сбрасываем непрочитанные
  useEffect(() => {
    markRead(chatId)
  }, [chatId])

  useEffect(() => {
    setMessages([])
    getMessages(chatId).then(data => {
      setMessages(Array.isArray(data) ? data : [])
    })
  }, [chatId])

  useEffect(() => { inputRef.current?.focus() }, [chatId])

  useEffect(() => {
    if (messagesRef.current) {
      messagesRef.current.scrollTop = messagesRef.current.scrollHeight
    }
  }, [messages])

  useEffect(() => {
    const off = onWSMessage(msg => {
      if (msg.event === 'message:new' && msg.data.chatId === chatId) {
        setMessages(prev => [...prev, msg.data])
        markRead(chatId)  // входящее в открытом чате — сразу читаем
      }
    })
    return off
  }, [chatId])

  async function send() {
    if (!text.trim()) return
    const trimmed = text.trim()
    setText('')

    try {
      const msg = await sendMessage(chatId, trimmed)
      setMessages(prev => [...prev, msg])
      notifyOwnMessage(chatId)  // ← поднимаем чат наверх
    } catch (e) {
      console.error('Failed to send message:', e)
      setText(trimmed)
    }
  }

  function formatTime(dateString: string) {
    return new Date(dateString).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }

  function formatFullDate(dateString: string) {
    const d = new Date(dateString)
    return d.toLocaleDateString('ru-RU') + ', ' +
           d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }

  function isSameDay(a: string, b: string) {
    const d1 = new Date(a), d2 = new Date(b)
    return d1.getFullYear() === d2.getFullYear() &&
           d1.getMonth() === d2.getMonth() &&
           d1.getDate() === d2.getDate()
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', maxWidth: 700, width: '100%' }}>
      <h3>Chat ID: {chatId}</h3>

      <div ref={messagesRef} style={{ flex: 1, overflowY: 'auto', border: '1px solid #ddd', padding: 12, marginBottom: 12 }}>
        {messages.map((m, index) => {
          const user = getSafeUser(m.senderId)
          const prev = messages[index - 1]
          const isGroupStart = !prev || prev.senderId !== m.senderId || !isSameDay(prev.createdAt, m.createdAt)

          if (isGroupStart) {
            return (
              <div key={m.id} style={styles.groupStartWrapper}>
                <img src={user.picture} style={styles.avatar} />
                <div style={styles.groupContent}>
                  <div style={styles.headerLine}>
                    <span style={styles.userName}>{user.name}</span>
                    <span style={styles.fullDate}>{formatFullDate(m.createdAt)}</span>
                  </div>
                  <div style={styles.messageText}>{m.text}</div>
                </div>
              </div>
            )
          }

          return (
            <div key={m.id} style={styles.groupStartWrapper}>
              <div style={styles.timeColumn}>
                <span style={styles.smallTime}>{formatTime(m.createdAt)}</span>
              </div>
              <div style={styles.groupContent}>
                <div style={styles.messageText}>{m.text}</div>
              </div>
            </div>
          )
        })}
      </div>

      <div style={{ display: 'flex', gap: 10 }}>
        <input
          ref={inputRef}
          style={{ flex: 1.5, padding: 10, fontSize: 16 }}
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
  groupStartWrapper: { display: 'flex', gap: 12, marginTop: 2 },
  avatar: { width: 40, height: 40, borderRadius: '50%', objectFit: 'cover' as const, flexShrink: 0 },
  groupContent: { flex: 1, display: 'flex', flexDirection: 'column' as const },
  headerLine: { display: 'flex', alignItems: 'baseline', gap: 8 },
  userName: { fontWeight: 600, fontSize: 15 },
  fullDate: { fontSize: 12, color: '#888' },
  messageText: { marginTop: 4, lineHeight: 1.4, wordBreak: 'break-word' as const },
  timeColumn: { width: 40, display: 'flex', justifyContent: 'center', alignItems: 'baseline', paddingTop: 4 },
  smallTime: { fontSize: 12, color: '#888', width: 40, flexShrink: 0, marginTop: 4, textAlign: 'center' as const },
}