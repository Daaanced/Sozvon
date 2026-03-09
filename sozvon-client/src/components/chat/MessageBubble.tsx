//sozvon-client\src\components\chat\MessageBubble.tsx

import { styles } from './chat.styles'
import type { Message } from './chat.types'
import type { User } from '../../api/users'
import { useState } from 'react'

type Props = {
  message: Message
  user: User
  isGroupStart: boolean
  onImageClick: (url: string) => void
  onReply: (message: Message) => void
  onForward: (message: Message) => void
  onScrollToMessage: (id: string) => void
  highlight: boolean
  setRef: (el: HTMLDivElement | null) => void
}

function isImage(mimeType: string) {
  return mimeType.startsWith('image/')
}

function isVideo(mimeType: string) {
  return mimeType.startsWith('video/')
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function formatTime(dateString: string) {
  return new Date(dateString).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function formatFullDate(dateString: string) {
  const d = new Date(dateString)
  return d.toLocaleDateString('ru-RU') + ', ' +
         d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

// Заменить строку export default function:
export default function MessageBubble({ message: m, user, isGroupStart, onImageClick, onReply, onForward, onScrollToMessage, highlight, setRef }: Props) {
  const imageAttachments = m.attachments?.filter(a => isImage(a.mimeType)) ?? []
  const videoAttachments = m.attachments?.filter(a => isVideo(a.mimeType)) ?? []
  const fileAttachments  = m.attachments?.filter(a => !isImage(a.mimeType) && !isVideo(a.mimeType)) ?? []
  const multipleImages   = imageAttachments.length > 1
  const [hovered, setHovered] = useState(false)

  return (
	<div
	ref={setRef}
	onMouseEnter={() => setHovered(true)}
	onMouseLeave={() => setHovered(false)}
style={{
	...styles.groupStartWrapper,
	background: highlight
		? '#c8f6f0'
		: hovered
		? '#c9daed'
		: undefined,
	transition: 'background 0.2s',
	position: 'relative',
	}}
	>

      {/* Левая колонка — аватар или время */}
      {isGroupStart ? (
        <img src={user.picture} style={styles.avatar} alt={user.name} />
      ) : (
        <div style={styles.timeColumn}>
          <span style={styles.smallTime}>{formatTime(m.createdAt)}</span>
        </div>
      )}

      {/* Правая колонка */}
      <div style={styles.groupContent}>
        {isGroupStart && (
          <div style={styles.headerLine}>
            <span style={styles.userName}>{user.name}</span>
            <span style={styles.fullDate}>{formatFullDate(m.createdAt)}</span>
          </div>
        )}

		{m.replyToMessage && (
		<div
			onClick={() => onScrollToMessage(m.replyToMessage!.id)}
			style={{ borderLeft: '3px solid #888', paddingLeft: 8, marginBottom: 4, opacity: 0.7, fontSize: 13, cursor: 'pointer' }}
		>
			↩ {m.replyToMessage.text.slice(0, 80)}{m.replyToMessage.text.length > 80 ? '...' : ''}
		</div>
		)}

		{m.forwardedFromId && (
		<div style={{ borderLeft: '3px solid #A4C7F0', paddingLeft: 8, marginBottom: 4, opacity: 0.7, fontSize: 13 }}>
			↪ Переслано
		</div>
		)}

        {m.text && (
          <div style={styles.messageText}>{m.text}</div>
        )}

        {/* Изображения */}
        {imageAttachments.length > 0 && (
          <div style={{
            ...styles.imageGrid,
            gridTemplateColumns: multipleImages ? 'repeat(auto-fill, 80px)' : '1fr'
          }}>
            {imageAttachments.map(att => (
              <img
                key={att.id}
                src={att.url}
                alt={att.fileName}
                title={att.fileName}
                style={multipleImages ? styles.gridImage : styles.singleImage}
                onClick={() => onImageClick(att.url)}
              />
            ))}
          </div>
        )}

        {/* Видео */}
        {videoAttachments.map(att => (
          <video
            key={att.id}
            src={att.url}
            controls
            style={styles.inlineVideo}
            title={att.fileName}
          />
        ))}

        {/* Файлы */}
        {fileAttachments.length > 0 && (
          <div style={styles.attachments}>
            {fileAttachments.map(att => (
              <div key={att.id}>
                <a
                  href={att.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  style={styles.fileLink}
                >
                  <span style={styles.fileIcon}>📎</span>
                  <span style={styles.fileName}>{att.fileName}</span>
                  <span style={styles.fileSize}>{formatBytes(att.size)}</span>
                </a>
              </div>
            ))}
          </div>
        )}

	{hovered && (
	<div style={{
		position: 'absolute',
		top: 4,
		right: 8,
		display: 'flex',
		gap: 4,
		background: 'rgba(148, 144, 144, 0.92)',
		borderRadius: 8,
		padding: '2px 6px',
		boxShadow: '0 2px 8px rgba(0,0,0,0.4)',
		zIndex: 10,
	}}>
		<button
		onClick={() => onReply(m)}
		title="Ответить"
		style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: 16, opacity: 0.85, padding: '2px 4px' }}
		>↩</button>
		<button
		onClick={() => onForward(m)}
		title="Переслать"
		style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: 16, opacity: 0.85, padding: '2px 4px' }}
		>↪</button>
	</div>
	)}
      </div>
    </div>
  )
}