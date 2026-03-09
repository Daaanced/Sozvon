//sozvon-client\src\components\chat\Chat.tsx
import { useEffect, useState, useCallback } from 'react'
import { onWSMessage } from '../../services/ws'
import { getMessages, getMessagesContext, sendMessage, uploadFiles } from '../../api/chats'
import { useChatContext } from '../../context/ChatContext'
import ChatMessages from './ChatMessages'
import ChatInput from './ChatInput'
import ChatTopBar from './ChatTopBar'
import Lightbox from './Lightbox'
import { styles } from './chat.styles'
import type { Message, PendingFile } from './chat.types'

type Props = { chatId: string }

export default function Chat({ chatId }: Props) {
  const { getSafeUser, markRead, notifyOwnMessage, chats, myId } = useChatContext()

  const [messages, setMessages] = useState<Message[]>([])
  const [text, setText] = useState('')
  const [pendingFiles, setPendingFiles] = useState<PendingFile[]>([])
  const [uploading, setUploading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState(0)
  const [lightboxUrl, setLightboxUrl] = useState<string | null>(null)
  const [replyTo, setReplyTo] = useState<Message | null>(null)
  const [forwardFrom, setForwardFrom] = useState<Message | null>(null)
  const [highlightId, setHighlightId] = useState<string | null>(null)
  // Находим собеседника по chatId
  const chat = chats.find(c => c.chatId === chatId)
  const withId = chat?.members.find(m => m !== myId)
  const companion = withId ? getSafeUser(withId) : null

  useEffect(() => {
    markRead(chatId)
    setMessages([])
    setPendingFiles([])
    setText('')
    getMessages(chatId).then(data => {
      setMessages(Array.isArray(data) ? data : [])
    })
  }, [chatId])

  useEffect(() => {
    const off = onWSMessage(msg => {
      if (msg.event === 'message:new' && msg.data.chatId === chatId) {
        setMessages(prev => [...prev, msg.data])
        markRead(chatId)
      }
    })
    return off
  }, [chatId])

  const handleFilesAdded = useCallback((newFiles: PendingFile[]) => {
    setPendingFiles(prev => [...prev, ...newFiles])
  }, [])

  const handleFileRemove = useCallback((index: number) => {
    setPendingFiles(prev => {
      const updated = [...prev]
      if (updated[index].previewUrl) URL.revokeObjectURL(updated[index].previewUrl!)
      updated.splice(index, 1)
      return updated
    })
  }, [])

// Заменить сигнатуру функции:
async function scrollToMessage(id: string, messageRefs: React.MutableRefObject<Record<string, HTMLDivElement | null>>) {
  // Сообщение уже загружено
  if (messageRefs.current[id]) {
    messageRefs.current[id]!.scrollIntoView({ behavior: 'smooth', block: 'center' })
    setHighlightId(id)
    setTimeout(() => setHighlightId(null), 1500)
    return
  }

  // Подгружаем контекст
  try {
    const data = await getMessagesContext(chatId, id)
    if (!Array.isArray(data)) return

    // Мержим — добавляем только те, которых ещё нет
    setMessages(prev => {
      const existingIds = new Set(prev.map(m => m.id))
      const newOnes = data.filter((m: Message) => !existingIds.has(m.id))
      const merged = [...prev, ...newOnes].sort(
        (a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()
      )
      return merged
    })

    // Скролл после рендера
    setHighlightId(id)
    setTimeout(() => {
      messageRefs.current[id]?.scrollIntoView({ behavior: 'smooth', block: 'center' })
      setTimeout(() => setHighlightId(null), 1500)
    }, 100)
  } catch (e) {
    console.error('Failed to load message context:', e)
  }
}

  async function send() {
    const trimmed = text.trim()
    const hasFiles = pendingFiles.length > 0
    if (!trimmed && !hasFiles) return

    setText('')

    if (hasFiles) {
      setUploading(true)
      setUploadProgress(0)
      const files = pendingFiles.map(pf => pf.file)
      pendingFiles.forEach(pf => { if (pf.previewUrl) URL.revokeObjectURL(pf.previewUrl) })
      setPendingFiles([])
      try {
        const msg = await uploadFiles(chatId, trimmed, files, replyTo?.id, forwardFrom?.id, setUploadProgress)
        setMessages(prev => [...prev, msg])
        notifyOwnMessage(chatId)
		setReplyTo(null)
		setForwardFrom(null)
      } catch (e) {
        console.error('Upload failed:', e)
        setText(trimmed)
      } finally {
        setUploading(false)
        setUploadProgress(0)
      }
    } else {
      try {
        const msg = await sendMessage(chatId, trimmed, replyTo?.id, forwardFrom?.id)
        setMessages(prev => [...prev, msg])
        notifyOwnMessage(chatId)
		setReplyTo(null)
		setForwardFrom(null)
      } catch (e) {
        console.error('Send failed:', e)
        setText(trimmed)
      }
    }
  }

  return (
    <div style={styles.chatWrapper}>
      {lightboxUrl && (
        <Lightbox url={lightboxUrl} onClose={() => setLightboxUrl(null)} />
      )}

      {/* Верхний бар */}
      <ChatTopBar
        user={companion}
        onCall={() => console.log('call', chatId)}
        onSettings={() => console.log('settings', chatId)}
      />

	<ChatMessages
		messages={messages}
		getUser={getSafeUser}
		onImageClick={setLightboxUrl}
		onReply={setReplyTo}
		onForward={setForwardFrom}
		highlightId={highlightId}             
		onScrollToMessage={scrollToMessage}
	/>

    <ChatInput
		text={text}
		pendingFiles={pendingFiles}
		uploading={uploading}
		uploadProgress={uploadProgress}
		replyTo={replyTo}
		forwardFrom={forwardFrom}
		onCancelReply={() => setReplyTo(null)}
		onCancelForward={() => setForwardFrom(null)}
		onTextChange={setText}
		onSend={send}
		onFilesAdded={handleFilesAdded}
		onFileRemove={handleFileRemove}
	/>
    </div>
  )
}