//sozvon-client\src\pages\ChatPage.tsx
import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import Chat from '../components/Chat'

export default function ChatPage() {
  const { login } = useParams()
  const token = localStorage.getItem('token')!
  const loginFromToken = parseToken(token)
  const [chatId, setChatId] = useState<string | null>(null)

  useEffect(() => {
    async function createChat() {
      const res = await fetch('http://90.189.252.24:8080/chats/create', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`
        },
        body: JSON.stringify({
          from: loginFromToken,
          to: login
        })
      })

      const chat = await res.json()
      setChatId(chat.id)
    }

    createChat()
  }, [login, token])

  if (!chatId) return <div>Loading chat...</div>

  return (
    <div>
      <h3>Chat with {login}</h3>
      <Chat chatId={chatId} token={token} />
    </div>
  )
}

function parseToken(token: string): string | null {
  try {
    const payload = token.split('.')[1] // берём вторую часть JWT
    const decoded = atob(payload.replace(/-/g, '+').replace(/_/g, '/'))
    const obj = JSON.parse(decoded)
    return obj.login || null
  } catch (e) {
    console.error("Invalid token", e)
    return null
  }
}
