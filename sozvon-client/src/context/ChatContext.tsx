//sozvon-client\src\context\ChatContext.tsx

import { createContext, useContext, useEffect, useState } from 'react'
import { searchUser, User } from '../api/users'
import { parseToken } from '../functions/parse'
import { onWSMessage } from '../services/ws'

type Chat = {
  chatId: string
  members: string[]
}

type ChatContextType = {
  chats: Chat[]
  users: Record<string, User>
  myLogin: string
  me: User | null
}


const ChatContext = createContext<ChatContextType | null>(null)

export function useChatContext() {
  const ctx = useContext(ChatContext)

  if (!ctx) {
    throw new Error('useChatContext must be used inside ChatProvider')
  }

  return ctx
}


export function ChatProvider({ children }: { children: React.ReactNode }) {
  const token = localStorage.getItem('token')
 
  if (!token) {
  throw new Error('No token found')
  }

  const myLogin = parseToken(token)!
  const [me, setMe] = useState<User | null>(null)
  const [chats, setChats] = useState<Chat[]>([])
  const [users, setUsers] = useState<Record<string, User>>({})

  async function loadChats() {
    const res = await fetch('http://176.51.121.88:8080/chats', {
      headers: { Authorization: `Bearer ${token}` }
    })

    if (!res.ok) return

    const data: Chat[] = await res.json()
    setChats(data)

    data.forEach(async chat => {
      const withLogin = chat.members.find(m => m !== myLogin)
      if (withLogin && !users[withLogin]) {
        const u = await searchUser(withLogin)
        setUsers(prev => ({ ...prev, [withLogin]: u }))
      }
    })
  }

useEffect(() => {
  if (!token || !myLogin) return

  loadChats()
  if (myLogin) {
  searchUser(myLogin).then(setMe)
  }


  const off = onWSMessage(msg => {
    if (msg.event === 'chat:created' || msg.event === 'message:new') {
      loadChats()
    }
  })

  return off
}, [token, myLogin])


  return (
    <ChatContext.Provider value={{ chats, users, myLogin, me }}>
      {children}
    </ChatContext.Provider>
  )
}