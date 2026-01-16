import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { login as loginRequest } from '../api/auth'

export default function Login() {
  const [login, setLogin] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const navigate = useNavigate()

  async function handleLogin() {
    try {
      const res = await loginRequest(login, password)
      localStorage.setItem('token', res.token)
      navigate('/main')
    } catch (e: any) {
      setError(e.message)
    }
  }

  return (
    <div>
      <h2>Login</h2>

      <input
        placeholder="Login"
        value={login}
        onChange={(e) => setLogin(e.target.value)}
      />

      <input
        placeholder="Password"
        type="password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
      />

      <button onClick={handleLogin}>Login</button>

      {error && <p style={{ color: 'red' }}>{error}</p>}

      <Link to="/register">Register</Link>
    </div>
  )
}
