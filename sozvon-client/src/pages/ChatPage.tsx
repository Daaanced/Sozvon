// sozvon-client/src/pages/ChatPage.tsx
import { useParams } from 'react-router-dom'
import Chat from '../components/Chat'

export default function ChatPage() {
  const { chatId } = useParams<{ chatId: string }>()

  if (!chatId) {
    return <div>Select a chat</div>
  }

  return <Chat chatId={chatId} />
}
