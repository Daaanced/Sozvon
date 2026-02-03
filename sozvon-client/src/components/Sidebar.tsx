// sozvon-client/src/components/Sidebar.tsx

import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { onWSMessage } from '../services/ws'
import { parseToken } from '../functions/parse'

type Chat = {
  chatId: string
  members: string[]
}

export default function Sidebar() {
  const token = localStorage.getItem('token')!
  const myLogin = parseToken(token)!
  const navigate = useNavigate()
  const [chats, setChats] = useState<Chat[]>([])

  function addChat(chat: Chat) {
    setChats(prev =>
      prev.find(c => c.chatId === chat.chatId)
        ? prev
        : [...prev, chat]
    )
  }

  useEffect(() => {
    const off = onWSMessage(msg => {
      if (msg.event === 'chat:created') {
        addChat({
          chatId: msg.data.chatId,
          members: msg.data.members
        })
      }

    //   if (msg.event === 'message:new') {
    //     addChat({
    //       chatId: msg.data.chatId,
    //       login: msg.data.from
    //     })
    //   }
    })

    return off
  }, [])

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Guild list */}
      <div>
        <h4>Guilds</h4>
        <div style={{ display: 'flex', gap: 8 }}>
          <div style={guildStyle}>G1</div>
          <div style={guildStyle}>G2</div>
          <div style={guildStyle}>+</div>
        </div>
      </div>

      {/* Navigation */}
      <div>
        <button onClick={() => navigate('/app')}>
          Search user
        </button>

        <button disabled style={{ marginLeft: 8 }}>
          Friends
        </button>
      </div>

      {/* Direct messages */}
      <div>
        <h4>Direct Messages</h4>

        {chats.map(chat => {
  const withLogin = chat.members.find(m => m !== myLogin)

		return (
			<div
			key={chat.chatId}
			style={dmStyle}
			onClick={() => navigate(`/app/chat/${chat.chatId}`)}
			>
			{withLogin}
			</div>
		)
		})}

      </div>
    </div>
  )
}

const guildStyle = {
  width: 40,
  height: 40,
  borderRadius: '50%',
  background: '#ddd',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  cursor: 'pointer'
}

const dmStyle = {
  padding: 8,
  borderRadius: 6,
  cursor: 'pointer',
  background: '#f3f3f3'
}
