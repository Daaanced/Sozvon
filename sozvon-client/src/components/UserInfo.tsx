//sozvon-client\src\components\UserInfo.tsx

import { useLocation } from 'react-router-dom'
import { useChatContext } from '../context/ChatContext'

export default function UserInfo() {
  const { chats, users, myLogin } = useChatContext()
  const location = useLocation()

  const chatId = location.pathname.split('/').pop()
  const chat = chats.find(c => c.chatId === chatId)

  if (!chat) {
    return <div style={{ padding: 16 }}>Select a chat</div>
  }

  const withLogin = chat.members.find(m => m !== myLogin)
  const user = withLogin ? users[withLogin] : null

  if (!user) {
    return <div style={{ padding: 16 }}>Loading...</div>
  }

  return (
    <div style={{ padding: 16 }}>
      <div style={{ marginBottom: 12 }}>
        {user.picture ? (
          <img
            src={user.picture}
            style={{
              width: 100,
              height: 100,
              borderRadius: '50%',
              objectFit: 'cover'
            }}
          />
        ) : (
          <div>{user.login[0].toUpperCase()}</div>
        )}
      </div>

      <div><b>Login:</b> {user.login}</div>
      <div><b>Name:</b> {user.name}</div>
      <div><b>Email:</b> {user.email || '-'}</div>
      <div><b>Info:</b> {user.info || '-'}</div>
      <div>
        <b>Created:</b> {new Date(user.created_at).toLocaleString()}
      </div>
    </div>
  )
}


// const styles = {
//   container: {
//     padding: 16,
//     display: 'flex',
//     flexDirection: 'column' as const,
//     gap: 10,
//   },
//   empty: {
//     padding: 16,
//     color: '#777'
//   },
//   avatar: {
//     width: 100,
//     height: 100,
//     borderRadius: '50%',
//     background: '#ddd',
//     overflow: 'hidden',
//     display: 'flex',
//     alignItems: 'center',
//     justifyContent: 'center',
//     marginBottom: 12
//   },
//   avatarImg: {
//     width: '100%',
//     height: '100%',
//     objectFit: 'cover' as const,
//   },
//   field: {
//     fontSize: 14
//   }
// }
