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
  getSafeUser: (login: string) => User
}

export const DELETED_USER: User = {
  id: 0,
  login: '-',
  name: 'deleted',	
  email: '-',
  info: '-',
  picture: 'http://92.127.177.190:8080/static/avatars/deleted.png',
  created_at: '-',
  updated_at: '-',
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
  return null
}

  const myLogin = parseToken(token)!
  const [me, setMe] = useState<User | null>(null)
  const [chats, setChats] = useState<Chat[]>([])
  const [users, setUsers] = useState<Record<string, User>>({})

  function getSafeUser(login: string): User {
  if (login === myLogin) {
    return me || DELETED_USER
  }

  return users[login] || DELETED_USER
}

  async function loadChats() {
  const res = await fetch('http://92.127.177.190:8080/chats', {
    headers: { Authorization: `Bearer ${token}` }
  })

  if (!res.ok) {
    setChats([])
    return
  }

  const data = await res.json()

  // 🔐 защита от null
  const safeChats: Chat[] = Array.isArray(data) ? data : []
  setChats(safeChats)

  safeChats.forEach(async chat => {
    const withLogin = chat.members?.find(m => m !== myLogin)
    if (withLogin && !users[withLogin]) {
  try {
    const u = await searchUser(withLogin)
    setUsers(prev => ({
      ...prev,
      [withLogin]: u || DELETED_USER
    }))
  } catch {
    setUsers(prev => ({
      ...prev,
      [withLogin]: DELETED_USER
    }))
  }
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
    <ChatContext.Provider value={{ chats, users, myLogin, me, getSafeUser }}>
      {children}
    </ChatContext.Provider>
  )
}