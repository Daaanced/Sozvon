// sozvon-client\src\layouts\AppShell.tsx
import { Outlet } from 'react-router-dom'
import Sidebar from '../components/Sidebar'
import UserInfo from '../components/UserInfo'
import { ChatProvider } from '../context/ChatContext'
import { connectWS } from '../services/ws'
import { useEffect } from 'react'

export default function AppShell() {
	const token = localStorage.getItem('token')

  	useEffect(() => {
    if (token) {
      connectWS(token)
    }
  }, [token])
  
  return (
    <ChatProvider>
      <div style={{ display: 'flex', height: '100%' }}>
        
        {/* LEFT */}
        <div style={{ width: 260, borderRight: '1px solid #ddd' }}>
          <Sidebar />
        </div>

        {/* CENTER */}
        <div
          style={{
            flex: 1,
            display: 'flex',
            justifyContent: 'center',
            padding: 12
          }}
        >
          <div style={{ width: 700, height: '100%' }}>
            <Outlet />
          </div>
        </div>

        {/* RIGHT */}
        <div
          style={{
            width: 300,
            borderLeft: '1px solid #ddd',
            background: '#fafafa'
          }}
        >
          <UserInfo />
        </div>

      </div>
    </ChatProvider>
  )
}
