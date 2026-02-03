// sozvon-client/src/pages/Main.tsx
import { useState } from 'react'
import { searchUser } from '../api/users'
import UserSearchResult from '../components/UserSearchResult'
import { useNavigate } from 'react-router-dom'
import { parseToken } from '../functions/parse'

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

  async function handleChat(userLogin: string) {
    try {
      const token = localStorage.getItem('token')!
      const loginFromToken = parseToken(token)

      const res = await fetch('http://90.189.252.24:8080/chats/create', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`
        },
        body: JSON.stringify({
          from: loginFromToken,
          to: userLogin
        })
      })

      const chat = await res.json()
      navigate(`/app/chat/${chat.id}`)
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
          onChat={() => handleChat(user.login)}
          onCall={() => console.log('Call', user.login)}
        />
      )}
    </div>
  )
}
