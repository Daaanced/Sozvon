//sozvon-client\src\router.tsx
import { createBrowserRouter } from 'react-router-dom'
import Login from './pages/Login'
import Register from './pages/Register'
import Main from './pages/Main'
import ChatPage from './pages/ChatPage'
import AppShell from './layouts/AppShell'

export const router = createBrowserRouter([
  { path: '/', element: <Login /> },
  { path: '/register', element: <Register /> },

  {
    path: '/app',
    element: <AppShell />,
    children: [
      {
        index: true,
        element: <Main />
      },
      {
        path: 'chat/:chatId',
  		element: <ChatPage />
      }
    ]
  }
])
