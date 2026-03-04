//sozvon-client\src\components\chat\ChatInput.tsx

import { useRef, useCallback } from 'react'
import { styles } from './chat.styles'
import type { PendingFile } from './chat.types'

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
  const inputRef = useRef<HTMLInputElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Фокус передаётся из Chat через ref — не нужен здесь
  // Оставляем публичный метод через forwardRef если понадобится

  const handlePaste = useCallback((e: React.ClipboardEvent) => {
    const items = Array.from(e.clipboardData.items)
    const imageItems = items.filter(i => i.kind === 'file' && i.type.startsWith('image/'))
    if (imageItems.length === 0) return

    e.preventDefault()
    const newFiles: PendingFile[] = imageItems.map(item => {
      const file = item.getAsFile()!
      return { file, previewUrl: URL.createObjectURL(file) }
    })
    onFilesAdded(newFiles)
  }, [onFilesAdded])

  function handleFileSelect(e: React.ChangeEvent<HTMLInputElement>) {
    const selected = Array.from(e.target.files || [])
    if (selected.length === 0) return

    const newFiles: PendingFile[] = selected.map(file => ({
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
          style={{ display: 'none' }}
          onChange={handleFileSelect}
        />

        <button
          style={styles.attachBtn}
          onClick={() => fileInputRef.current?.click()}
          disabled={uploading}
          title="Attach files"
        >
          📎
        </button>

        <input
          ref={inputRef}
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
          style={styles.sendBtn}
          onClick={onSend}
          disabled={uploading || (!text.trim() && pendingFiles.length === 0)}
        >
          {uploading ? '...' : 'Send'}
        </button>
      </div>
    </div>
  )
}