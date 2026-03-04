// sozvon-client/src/context/ChatContext.tsx
import { createContext, useContext, useEffect, useState } from 'react'
import { getUserById, searchUser, User } from '../api/users'
import { onWSMessage } from '../services/ws'
import { requestAuth } from '../api/http'

type Chat = {
  chatId: string
  members: number[]
  lastMessage?: string
  updatedAt?: string
}

type ChatContextType = {
  chats: Chat[]
  users: Record<number, User>
  myId: number
  myLogin: string
  me: User | null
  unread: Record<string, boolean>  // chatId -> есть ли непрочитанные
  markRead: (chatId: string) => void
  notifyOwnMessage: (chatId: string) => void  // вызывается из Chat.tsx после отправки
  getSafeUser: (id: number) => User
}

export const DELETED_USER: User = {
  id: 0,
  login: '-',
  name: 'Deleted',
  email: '-',
  info: '-',
  picture: 'http://92.127.177.190:8080/static/avatars/deleted.png',
  created_at: '-',
  updated_at: '-',
}

const ChatContext = createContext<ChatContextType | null>(null)

export function useChatContext() {
  const ctx = useContext(ChatContext)
  if (!ctx) throw new Error('useChatContext must be used inside ChatProvider')
  return ctx
}

function parseTokenPayload(token: string): { id: number; login: string } | null {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    const id = parseInt(payload.user_id, 10)
    const login = payload.login ?? ''
    if (!id) return null
    return { id, login }
  } catch {
    return null
  }
}

// Сортировка чатов по updatedAt (свежие сверху)
function sortChats(chats: Chat[]): Chat[] {
  return [...chats].sort((a, b) => {
    const ta = a.updatedAt ? new Date(a.updatedAt).getTime() : 0
    const tb = b.updatedAt ? new Date(b.updatedAt).getTime() : 0
    return tb - ta
  })
}

export function ChatProvider({ children }: { children: React.ReactNode }) {
  const token = localStorage.getItem('token')
  if (!token) return null

  const parsed = parseTokenPayload(token)
  if (!parsed) return null

  const { id: myId, login: myLogin } = parsed

  const [me, setMe] = useState<User | null>(null)
  const [chats, setChats] = useState<Chat[]>([])
  const [users, setUsers] = useState<Record<number, User>>({})
  const [unread, setUnread] = useState<Record<string, boolean>>({})

  function getSafeUser(id: number): User {
    if (id === myId) return me ?? DELETED_USER
    return users[id] ?? DELETED_USER
  }

  function markRead(chatId: string) {
    setUnread(prev => ({ ...prev, [chatId]: false }))
  }

  // Вызывается из Chat.tsx после успешной отправки своего сообщения
  function notifyOwnMessage(chatId: string) {
    setChats(prev => {
      const now = new Date().toISOString()
      const updated = prev.map(c =>
        c.chatId === chatId ? { ...c, updatedAt: now } : c
      )
      return sortChats(updated)
    })
  }

  function loadUserById(id: number) {
    if (id === myId) return
    setUsers(prev => {
      if (prev[id]) return prev
      getUserById(id)
        .then(u => setUsers(p => ({ ...p, [id]: u ?? DELETED_USER })))
        .catch(() => setUsers(p => ({ ...p, [id]: DELETED_USER })))
      return prev
    })
  }

  async function loadChats() {
    try {
      const data = await requestAuth('/chats')
      const safeChats: Chat[] = Array.isArray(data) ? data : []
      const sorted = sortChats(safeChats)
      setChats(sorted)
      sorted.forEach(chat => {
        chat.members.filter(id => id !== myId).forEach(loadUserById)
      })
    } catch {
      setChats([])
    }
  }

  useEffect(() => {
    searchUser(myLogin).then(setMe).catch(() => {})
    loadChats()

    const off = onWSMessage(msg => {
      if (msg.event === 'chat:created' || msg.event === 'chat:activated') {
        loadChats()
        return
      }

      if (msg.event === 'message:new') {
        const chatId: string = msg.data.chatId

        // Поднимаем чат наверх и обновляем updatedAt
        setChats(prev => {
          const now = msg.data.createdAt ?? new Date().toISOString()
          const updated = prev.map(c =>
            c.chatId === chatId ? { ...c, updatedAt: now } : c
          )
          return sortChats(updated)
        })

        // Помечаем как непрочитанное (Sidebar проверит, открыт ли чат)
        setUnread(prev => ({ ...prev, [chatId]: true }))
      }
    })

    return off
  }, [])

  return (
    <ChatContext.Provider value={{
      chats, users, myId, myLogin, me,
      unread, markRead, notifyOwnMessage,
      getSafeUser
    }}>
      {children}
    </ChatContext.Provider>
  )
}