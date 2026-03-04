//sozvon-client\src\components\chat\MessageBubble.tsx

import { styles } from './chat.styles'
import type { Message } from './chat.types'
import type { User } from '../../api/users'

type Props = {
  message: Message
  user: User
  isGroupStart: boolean
  onImageClick: (url: string) => void
}

function isImage(mimeType: string) {
  return mimeType.startsWith('image/')
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

export default function MessageBubble({ message: m, user, isGroupStart, onImageClick }: Props) {
  return (
    <div style={styles.groupStartWrapper}>
      {/* Левая колонка — аватар или время */}
      {isGroupStart ? (
        <img src={user.picture} style={styles.avatar} alt={user.name} />
      ) : (
        <div style={styles.timeColumn}>
          <span style={styles.smallTime}>{formatTime(m.createdAt)}</span>
        </div>
      )}

      {/* Правая колонка — контент */}
      <div style={styles.groupContent}>
        {isGroupStart && (
          <div style={styles.headerLine}>
            <span style={styles.userName}>{user.name}</span>
            <span style={styles.fullDate}>{formatFullDate(m.createdAt)}</span>
          </div>
        )}

        {m.text && (
          <div style={styles.messageText}>{m.text}</div>
        )}

        {m.attachments && m.attachments.length > 0 && (
  <div style={styles.attachments}>
    {m.attachments.map(att => (
      <div key={att.id}>
        {isImage(att.mimeType) ? (
          <img
            src={att.url}
            alt={att.fileName}
            style={styles.inlineImage}
            onClick={() => onImageClick(att.url)}
            title={att.fileName}
          />
        ) : (
		<a
			href={att.url}
			download={att.fileName}
			style={styles.fileLink}
		>
			<span style={styles.fileIcon}>📎</span>
			<span style={styles.fileName}>{att.fileName}</span>
			<span style={styles.fileSize}>{formatBytes(att.size)}</span>
		</a>
)}
      </div>
    ))}
  </div>
)}
      </div>
    </div>
  )
}