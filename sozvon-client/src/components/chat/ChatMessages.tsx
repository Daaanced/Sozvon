import { useRef, useEffect } from 'react'
import MessageBubble from './MessageBubble'
import { styles } from './chat.styles'
import type { Message } from './chat.types'
import type { User } from '../../api/users'

type Props = {
  messages: Message[]
  getUser: (senderId: number) => User
  onImageClick: (url: string) => void
}

function isSameDay(a: string, b: string) {
  const d1 = new Date(a), d2 = new Date(b)
  return d1.getFullYear() === d2.getFullYear() &&
         d1.getMonth() === d2.getMonth() &&
         d1.getDate() === d2.getDate()
}

export default function ChatMessages({ messages, getUser, onImageClick }: Props) {
  const ref = useRef<HTMLDivElement>(null)

  // Автоскролл вниз при новых сообщениях
  useEffect(() => {
    if (ref.current) {
      ref.current.scrollTop = ref.current.scrollHeight
    }
  }, [messages])

  return (
    <div ref={ref} style={styles.messageList}>
      {messages.map((m, index) => {
        const prev = messages[index - 1]
        const isGroupStart =
          !prev ||
          prev.senderId !== m.senderId ||
          !isSameDay(prev.createdAt, m.createdAt)

        return (
          <MessageBubble
            key={m.id}
            message={m}
            user={getUser(m.senderId)}
            isGroupStart={isGroupStart}
            onImageClick={onImageClick}
          />
        )
      })}
    </div>
  )
}