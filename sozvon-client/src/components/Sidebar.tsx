// sozvon-client/src/components/Sidebar.tsx
import { useNavigate, useLocation } from 'react-router-dom'
import { useChatContext } from '../context/ChatContext'
import { useState } from 'react'
import SettingsModal from './SettingsModal'

export default function Sidebar() {
  const { chats, users, myLogin, me } = useChatContext()

  const navigate = useNavigate()
  const location = useLocation()
  const [open, setOpen] = useState(false)
  return (
    <div style={styles.sidebar}>
      <div
        style={styles.searchButton}
        onClick={() => navigate('/app')}
      >
        🔍 Search users
      </div>

      <div style={styles.chatList}>
        {chats.map(chat => {
          const withLogin = chat.members.find(m => m !== myLogin)!
          const user = users[withLogin]
          const isActive = location.pathname.endsWith(chat.chatId)

          return (
            <div
              key={chat.chatId}
              onClick={() => navigate(`/app/chats/${chat.chatId}`)}
              style={{
                ...styles.chatItem,
                background: isActive ? '#e0e0ff' : '#fff',
              }}
            >
              <div style={styles.avatar}>
                {user?.picture
                  ? <img src={user.picture} style={styles.avatarImg} />
                  : <span>{withLogin[0].toUpperCase()}</span>
                }
              </div>
              <span>{user?.name || withLogin}</span>
            </div>
          )
        })}
      </div>
	  <div style={styles.meBlock}>
  <div style={styles.meInfo}>
    <div style={styles.avatar}>
      {me?.picture
        ? <img src={me.picture} style={styles.avatarImg} />
        : <span>{myLogin[0].toUpperCase()}</span>
      }
    </div>
    <span>{me?.name || myLogin}</span>
  </div>

  <div style={styles.meButtons}>
    <button style={styles.iconBtn}>🎙️</button>
    <button style={styles.iconBtn}>🎧</button>
    <button
  style={styles.iconBtn}
  onClick={() => setOpen(true)}
>
  ⚙️
</button>

{open && (
  <SettingsModal onClose={() => setOpen(false)} />
)}
  </div>
</div>

    </div>
	
  )
}

const styles = {
  sidebar: {
    width: 260,
    height: '100vh',
    display: 'flex',
    flexDirection: 'column' as const,
    borderRight: '1px solid #ddd',
    background: '#f9f9f9',
    padding: 8,
    boxSizing: 'border-box' as const,
  },

  searchButton: {
    padding: '8px 10px',
    borderRadius: 6,
    background: '#fff',
    cursor: 'pointer',
    marginBottom: 8,
    textAlign: 'center' as const,
    fontWeight: 500,
  },

  chatList: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column' as const,
    gap: 6,
    overflowY: 'auto' as const,
  },

  chatItem: {
    display: 'flex',
    alignItems: 'center',
    gap: 10,
    padding: '6px 8px',
    borderRadius: 6,
    cursor: 'pointer',
  },

  avatar: {
    width: 36,
    height: 36,
    borderRadius: '50%',
    background: '#ddd',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    overflow: 'hidden',
    flexShrink: 0,
  },

  avatarImg: {
    width: '100%',
    height: '100%',
    objectFit: 'cover' as const,
  },

  meBlock: {
    borderTop: '1px solid #ddd',
    paddingTop: 8,
    marginTop: 8,
  },

  meInfo: {
    display: 'flex',
    alignItems: 'center',
    gap: 10,
    marginBottom: 6,
  },

  meButtons: {
    display: 'flex',
    justifyContent: 'space-between',
  },

  iconBtn: {
    flex: 1,
    margin: 2,
    padding: 6,
    borderRadius: 6,
    border: 'none',
    cursor: 'pointer',
  },
}
