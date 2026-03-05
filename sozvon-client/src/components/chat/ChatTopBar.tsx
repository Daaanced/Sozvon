//sozvon-client\src\components\chat\ChatTopBar.tsx

import { useState } from 'react'
import type { User } from '../../api/users'
import { styles } from './chat.styles'

type Props = {
  user: User | null
  onCall: () => void
  onSettings: () => void
}

export default function ChatTopBar({ user, onCall, onSettings }: Props) {
  const [searchOpen, setSearchOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')

  return (
    <div style={styles.bar}>

      {/* Левая часть — аватар + имя собеседника */}
      <div style={styles.userInfo}>
        {user?.picture && (
          <img src={user.picture} style={styles.avatar} alt={user.name} />
        )}
        <span style={styles.userName}>{user?.name ?? '...'}</span>
      </div>

      {/* Правая часть — действия */}
      <div style={styles.actions}>

        {/* Поиск — разворачивается при клике */}
        {searchOpen && (
          <input
            autoFocus
            style={styles.searchInput}
            placeholder="Search messages..."
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            onKeyDown={e => e.key === 'Escape' && setSearchOpen(false)}
          />
        )}

        <button
          style={styles.iconBtn}
          onClick={() => setSearchOpen(prev => !prev)}
          title="Search"
        >
          🔍
        </button>

        <button
          style={styles.iconBtn}
          onClick={onCall}
          title="Call"
        >
          📞
        </button>

        <button
          style={styles.iconBtn}
          onClick={onSettings}
          title="Settings"
        >
          ⚙️
        </button>

      </div>
    </div>
  )
}