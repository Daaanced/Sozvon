// sozvon-client/src/components/UserInfo.tsx
import { useLocation } from 'react-router-dom'
import { useChatContext, DELETED_USER } from '../context/ChatContext'

export default function UserInfo() {
  const { chats, myId, getSafeUser } = useChatContext()
  const location = useLocation()

  const chatId = location.pathname.split('/').pop()
  const chat = chats.find(c => c.chatId === chatId)

  if (!chat) {
    return <div style={{ padding: 16 }}>Select a chat</div>
  }

  const withId = chat.members.find(m => m !== myId)
  // Если собеседник не найден — показываем DELETED_USER
  const user = withId ? getSafeUser(withId) : DELETED_USER

  return (
    <div style={{ padding: 16 }}>
      <div style={{ marginBottom: 12 }}>
        {user.picture ? (
          <img
            src={user.picture}
            style={{ width: 100, height: 100, borderRadius: '50%', objectFit: 'cover' }}
          />
        ) : (
          <div>{user.login[0].toUpperCase()}</div>
        )}
      </div>

      <div><b>Login:</b> {user.login}</div>
      <div><b>Name:</b> {user.name}</div>
      <div><b>Email:</b> {user.email || '-'}</div>
      <div><b>Info:</b> {user.info || '-'}</div>
      <div><b>Created:</b> {new Date(user.created_at).toLocaleString()}</div>
    </div>
  )
}