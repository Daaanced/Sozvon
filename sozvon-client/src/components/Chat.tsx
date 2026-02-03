// sozvon-client/src/components/Chat.tsx

import { useEffect, useState } from 'react'
import { onWSMessage, sendWS } from '../services/ws'

type Props = {
  chatId: string
}

export default function Chat({ chatId }: Props) {
  const [messages, setMessages] = useState<any[]>([])
  const [text, setText] = useState('')

  useEffect(() => {
    const off = onWSMessage(msg => {
      if (msg.event === 'message:new' && msg.data.chatId === chatId) {
        setMessages(prev => [
          ...prev,
          { from: msg.data.from, text: msg.data.text }
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
      {messages.map((m, i) => (
        <div key={i}>
          <b>{m.from}:</b> {m.text}
        </div>
      ))}

      <input
        value={text}
        onChange={e => setText(e.target.value)}
        onKeyDown={e => {
          if (e.key === 'Enter') send()
        }}
      />
    </div>
  )
}