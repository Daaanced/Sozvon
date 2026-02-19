//sozvon-client\src\pages\Register.tsx
import { useState } from 'react'
import { Link } from 'react-router-dom'
import { register as registerRequest } from '../api/auth'

export default function Register() {
  const [login, setLogin] = useState('')
  const [password, setPassword] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  async function handleRegister() {
    setMessage('')
    setError('')

    if (!login || !password) {
      setError('Login and password are required')
      return
    }

    try {
      await registerRequest(login, password)
      setMessage('Registration successful. You can now log in.')
      setLogin('')
      setPassword('')
    } catch (e: any) {
      setError(e.message || 'Registration failed')
    }
  }

  return (
    <div>
      <h2>Register</h2>

      <input
        placeholder="Login"
        value={login}
        onChange={(e) => setLogin(e.target.value)}
      />

      <input
        type="password"
        placeholder="Password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
      />

      <button onClick={handleRegister}>Register</button>

      {message && <p style={{ color: 'green' }}>{message}</p>}
      {error && <p style={{ color: 'red' }}>{error}</p>}

      <Link to="/login">Back to login</Link>
    </div>
  )
}
