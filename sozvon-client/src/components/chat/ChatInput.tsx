//sozvon-client\src\components\chat\ChatInput.tsx

import { useRef, useCallback } from 'react'
import { styles } from './chat.styles'
import type { PendingFile } from './chat.types'

const MAX_FILES = 4

type Props = {
  text: string
  pendingFiles: PendingFile[]
  uploading: boolean
  uploadProgress: number
  onTextChange: (value: string) => void
  onSend: () => void
  onFilesAdded: (files: PendingFile[]) => void
  onFileRemove: (index: number) => void
}

export default function ChatInput({
  text, pendingFiles, uploading, uploadProgress,
  onTextChange, onSend, onFilesAdded, onFileRemove
}: Props) {
  const fileInputRef = useRef<HTMLInputElement>(null)

  const remaining = MAX_FILES - pendingFiles.length

const handlePaste = useCallback((e: React.ClipboardEvent) => {
  const items = Array.from(e.clipboardData.items)

  // 1. Сначала проверяем HTML — там может быть прямая ссылка на GIF
  const htmlItem = items.find(i => i.kind === 'string' && i.type === 'text/html')
  if (htmlItem && remaining > 0) {
    htmlItem.getAsString(async (html) => {
      const match = html.match(/<img[^>]+src="([^"]+\.gif[^"]*)"/)
      if (!match) return

      e.preventDefault()
      try {
        const response = await fetch(match[1])
        const blob = await response.blob()
        const file = new File(
          [blob],
          `gif-${Date.now()}.gif`,
          { type: 'image/gif' }
        )
        onFilesAdded([{ file, previewUrl: URL.createObjectURL(file) }])
      } catch {
        console.warn('Failed to fetch GIF:', match[1])
      }
    })
    // Выходим — GIF обрабатывается асинхронно выше
    // PNG превью от Windows игнорируем
    return
  }

  // 2. Обычные файлы — скриншоты, скопированные изображения
  const fileItems = items.filter(
    i => i.kind === 'file' && i.type.startsWith('image/')
  )
  if (fileItems.length === 0 || remaining <= 0) return

  e.preventDefault()
  const toAdd = fileItems.slice(0, remaining)
  const newFiles: PendingFile[] = toAdd.map(item => {
    const file = item.getAsFile()!
    return { file, previewUrl: URL.createObjectURL(file) }
  })
  onFilesAdded(newFiles)

}, [onFilesAdded, remaining])

  function handleFileSelect(e: React.ChangeEvent<HTMLInputElement>) {
    const selected = Array.from(e.target.files || [])
    if (selected.length === 0) return
    if (remaining <= 0) return

    const toAdd = selected.slice(0, remaining)
    const newFiles: PendingFile[] = toAdd.map(file => ({
      file,
      previewUrl: file.type.startsWith('image/') ? URL.createObjectURL(file) : null
    }))

    onFilesAdded(newFiles)
    e.target.value = ''
  }

  return (
    <div>
      {/* Превью выбранных файлов */}
      {pendingFiles.length > 0 && (
        <div style={styles.pendingFiles}>
          {pendingFiles.map((pf, i) => (
            <div key={i} style={styles.pendingItem}>
              {pf.previewUrl ? (
                <img src={pf.previewUrl} style={styles.pendingPreview} alt="" />
              ) : (
                <div style={styles.pendingFileIcon}>📎</div>
              )}
              <span style={styles.pendingFileName}>
                {pf.file.name.length > 16
                  ? pf.file.name.slice(0, 13) + '...'
                  : pf.file.name}
              </span>
              <button style={styles.pendingRemove} onClick={() => onFileRemove(i)}>
                ✕
              </button>
            </div>
          ))}

          {/* Счётчик оставшихся слотов */}
          {pendingFiles.length > 0 && (
            <div style={styles.fileCounter}>
              {pendingFiles.length}/{MAX_FILES}
            </div>
          )}
        </div>
      )}

      {/* Прогресс */}
      {uploading && (
        <div style={styles.progressWrapper}>
          <div style={{ ...styles.progressFill, width: `${uploadProgress}%` }} />
          <span style={styles.progressText}>{uploadProgress}%</span>
        </div>
      )}

      {/* Строка ввода */}
      <div style={styles.inputRow}>
        <input
          ref={fileInputRef}
          type="file"
          multiple
          accept="image/*,video/*,application/pdf,.doc,.docx,.xls,.xlsx,.zip,.rar,.txt"
          style={{ display: 'none' }}
          onChange={handleFileSelect}
        />

        <button
          style={{
            ...styles.attachBtn,
            opacity: remaining <= 0 ? 0.4 : 1
          }}
          onClick={() => remaining > 0 && fileInputRef.current?.click()}
          disabled={uploading || remaining <= 0}
          title={remaining <= 0 ? `Max ${MAX_FILES} files` : 'Attach files'}
        >
          📎
        </button>

        <input
          style={styles.textInput}
          value={text}
          maxLength={4000}
          placeholder={pendingFiles.length > 0 ? 'Add a caption...' : 'Type message...'}
          disabled={uploading}
          onChange={e => onTextChange(e.target.value)}
          onKeyDown={e => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault()
              onSend()
            }
          }}
          onPaste={handlePaste}
        />

        <button
          style={{
            ...styles.sendBtn,
            opacity: (uploading || (!text.trim() && pendingFiles.length === 0)) ? 0.5 : 1
          }}
          onClick={onSend}
          disabled={uploading || (!text.trim() && pendingFiles.length === 0)}
        >
          {uploading ? '...' : 'Send'}
        </button>
      </div>
    </div>
  )
}