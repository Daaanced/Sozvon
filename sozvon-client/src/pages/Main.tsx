//sozvon-client\src\pages\Main.tsx
import { useState } from 'react'
import { searchUser } from '../api/users'
import UserSearchResult from '../components/UserSearchResult'
import { useNavigate } from 'react-router-dom'


export default function Main() {
  const [query, setQuery] = useState('')
  const [user, setUser] = useState<any>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()


  async function handleSearch() {
    if (!query) return

    setLoading(true)
    setError('')
    setUser(null)

    try {
      const result = await searchUser(query)
      setUser(result)
    } catch (e: any) {
      setError(e.message || 'User not found')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div
      style={{
        height: '100vh',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        paddingTop: 20
      }}
    >
      {/* Search input */}
      <div style={{ display: 'flex', gap: 8 }}>
        <input
          placeholder="Search user by login"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
        />
        <button onClick={handleSearch}>Search</button>
      </div>

      {loading && <p>Searching...</p>}
      {error && <p style={{ color: 'red' }}>{error}</p>}

      {user && (
        <UserSearchResult
  			login={user.login}
  			picture={user.picture}
  			onChat={() => navigate(`/chat/${user.login}`)}
  			onCall={() => console.log('Start call with', user.login)}
/>
      )}
    </div>
  )
}

