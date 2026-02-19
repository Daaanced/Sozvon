// sozvon-client/src/components/SettingsModal.tsx
import { useState } from "react"
import { useChatContext } from "../context/ChatContext"
import { updateUser, uploadAvatar } from '../api/users'

type Props = {
  onClose: () => void
}

export default function SettingsModal({ onClose }: Props) {
  const { me } = useChatContext()

  // локальное состояние для редактирования
  const [name, setName] = useState(me?.name || "")
  const [email, setEmail] = useState(me?.email || "")
  const [info, setInfo] = useState(me?.info || "")
  const [picture, setPicture] = useState<File | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")

 async function handleSave() {
  if (!me) return
  setLoading(true)
  setError("")

  try {
    // обновляем текстовые поля
    await updateUser(me.login, { name, email, info })

    // загружаем аватар отдельно, если выбран
    if (picture) {
      const res = await uploadAvatar(me.login, picture)
      console.log("Avatar URL:", res.avatar_url)
    }

    onClose()
  } catch (e: any) {
    setError(e.message || "Ошибка при сохранении")
  } finally {
    setLoading(false)
  }
}

  return (
    <div style={overlayStyle}>
      <div style={modalStyle}>
        <h2>Settings</h2>

        {/* === Вкладки, пока только Profile === */}
        <h3>Profile</h3>
        <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
          <input
            placeholder="Name"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <input
            placeholder="Email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
          <textarea
            placeholder="Info"
            value={info}
            onChange={(e) => setInfo(e.target.value)}
          />
          <input
            type="file"
            accept="image/*"
            onChange={(e) => {
              if (e.target.files && e.target.files[0]) {
                setPicture(e.target.files[0])
              }
            }}
          />
        </div>

        {error && <p style={{ color: "red" }}>{error}</p>}

        <div style={{ marginTop: 12, display: "flex", justifyContent: "flex-end", gap: 8 }}>
          <button onClick={onClose} disabled={loading}>Cancel</button>
          <button onClick={handleSave} disabled={loading}>{loading ? "Saving..." : "Save"}</button>
        </div>
      </div>
    </div>
  )
}

const overlayStyle: React.CSSProperties = {
  position: 'fixed',
  top: 0,
  left: 0,
  width: '100vw',
  height: '100vh',
  background: 'rgba(0,0,0,0.5)',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  zIndex: 999
}

const modalStyle: React.CSSProperties = {
  background: '#fff',
  padding: 20,
  borderRadius: 8,
  width: 400,
  maxHeight: '80vh',
  overflowY: 'auto'
}
