//sozvon-client\src\layouts\AppShell.tsx

import { Outlet } from 'react-router-dom'
import Sidebar from '../components/Sidebar'
import UserInfo from '../components/UserInfo'
import { ChatProvider } from '../context/ChatContext'
import { connectWS } from '../services/ws'
import { useEffect, useState } from 'react'

const USERINFO_BREAKPOINT = 1100

export default function AppShell() {
  const [width, setWidth] = useState(window.innerWidth)

  useEffect(() => {
    const token = localStorage.getItem('token')
    if (token) connectWS(token)
  }, [])

  useEffect(() => {
    const handleResize = () => setWidth(window.innerWidth)
    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [])

  const showUserInfo = width >= USERINFO_BREAKPOINT

  return (
    <ChatProvider>
      <div style={{ display: 'flex', height: '100vh', overflow: 'hidden' }}>

        {/* SIDEBAR */}
        <div style={{ width: 260, flexShrink: 0, borderRight: '1px solid #ddd' }}>
          <Sidebar />
        </div>

        {/* CENTER */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', padding: 12 }}>
          <Outlet />
        </div>

        {/* USER INFO */}
        {showUserInfo && (
          <div style={{ width: 300, flexShrink: 0, borderLeft: '1px solid #ddd', background: '#fafafa', overflowY: 'auto' }}>
            <UserInfo />
          </div>
        )}

      </div>
    </ChatProvider>
  )
}