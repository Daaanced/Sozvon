import { useEffect, useState, useCallback } from 'react'
import { onWSMessage } from '../../services/ws'
import { getMessages, sendMessage, uploadFiles } from '../../api/chats'
import { useChatContext } from '../../context/ChatContext'
import ChatMessages from './ChatMessages'
import ChatInput from './ChatInput'
import Lightbox from './Lightbox'
import { styles } from './chat.styles'
import type { Message, PendingFile } from './chat.types'

type Props = { chatId: string }

export default function Chat({ chatId }: Props) {
  const { getSafeUser, markRead, notifyOwnMessage } = useChatContext()

  const [messages, setMessages] = useState<Message[]>([])
  const [text, setText] = useState('')
  const [pendingFiles, setPendingFiles] = useState<PendingFile[]>([])
  const [uploading, setUploading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState(0)
  const [lightboxUrl, setLightboxUrl] = useState<string | null>(null)

  // Сброс при смене чата
  useEffect(() => {
    markRead(chatId)
    setMessages([])
    setPendingFiles([])
    setText('')

    getMessages(chatId).then(data => {
      setMessages(Array.isArray(data) ? data : [])
    })
  }, [chatId])

  // WS push
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
      if (updated[index].previewUrl) {
        URL.revokeObjectURL(updated[index].previewUrl!)
      }
      updated.splice(index, 1)
      return updated
    })
  }, [])

  async function send() {
    const trimmed = text.trim()
    const hasFiles = pendingFiles.length > 0
    if (!trimmed && !hasFiles) return

    setText('')

    if (hasFiles) {
      setUploading(true)
      setUploadProgress(0)

      const files = pendingFiles.map(pf => pf.file)
      pendingFiles.forEach(pf => {
        if (pf.previewUrl) URL.revokeObjectURL(pf.previewUrl)
      })
      setPendingFiles([])

      try {
        const msg = await uploadFiles(chatId, trimmed, files, setUploadProgress)
        setMessages(prev => [...prev, msg])
        notifyOwnMessage(chatId)
      } catch (e) {
        console.error('Upload failed:', e)
        setText(trimmed)
      } finally {
        setUploading(false)
        setUploadProgress(0)
      }
    } else {
      try {
        const msg = await sendMessage(chatId, trimmed)
        setMessages(prev => [...prev, msg])
        notifyOwnMessage(chatId)
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

      <ChatMessages
        messages={messages}
        getUser={getSafeUser}
        onImageClick={setLightboxUrl}
      />

      <ChatInput
        text={text}
        pendingFiles={pendingFiles}
        uploading={uploading}
        uploadProgress={uploadProgress}
        onTextChange={setText}
        onSend={send}
        onFilesAdded={handleFilesAdded}
        onFileRemove={handleFileRemove}
      />
    </div>
  )
}