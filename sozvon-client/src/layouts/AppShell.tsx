// sozvon-client\src\layouts\AppShell.tsx
import { useEffect } from 'react'
import { Outlet } from 'react-router-dom'
import Sidebar from '../components/Sidebar'
import { connectWS } from '../services/ws'

export default function AppShell() {
	const token = localStorage.getItem('token')

	useEffect(() => {
  	if (token) {
    	connectWS(token)
  	}
	}, [token])

  return (
    <div style={{ display: 'flex', height: '100vh' }}>
      {/* LEFT — всегда */}
      <div
        style={{
          width: 280,
          borderRight: '1px solid #ddd',
          padding: 12
        }}
      >
        <Sidebar />
      </div>

      {/* RIGHT — меняется */}
      <div style={{ flex: 1, padding: 12 }}>
        <Outlet />
      </div>
    </div>
  )
}
