// sozvon-client/src/pages/Main.tsx
import { useState } from 'react'
import { searchUser } from '../api/users'
import UserSearchResult from '../components/UserSearchResult'
import { useNavigate } from 'react-router-dom'
import { createChat } from '../api/chats'

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

  async function handleChat(toUserId: number) {
    try {
      const chat = await createChat(toUserId)
      navigate(`/app/chats/${chat.id}`)
    } catch (err) {
      console.error(err)
    }
  }

  return (
    <div>
      <h3>Search user</h3>
      <input
        placeholder="Login"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
      />
      <button onClick={handleSearch}>Search</button>
      {loading && <p>Searching...</p>}
      {error && <p style={{ color: 'red' }}>{error}</p>}
      {user && (
        <UserSearchResult
          login={user.login}
          picture={user.picture}
          onChat={() => handleChat(user.id)}  // ← передаём user.id (int)
          onCall={() => console.log('Call', user.login)}
        />
      )}
    </div>
  )
}