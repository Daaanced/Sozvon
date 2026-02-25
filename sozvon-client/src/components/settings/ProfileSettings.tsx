//sozvon-client\src\components\settings\ProfileSettings.tsx

import { useRef, useState } from "react"
import { useChatContext } from "../../context/ChatContext"
import {
  updateUser,
  uploadAvatar,
  deleteAvatar,
  UpdateUserPayload
} from "../../api/users"

// type Props = {
//   onClose: () => void
// }

export default function ProfileSettings() {
  const { me } = useChatContext()

  const fileInputRef = useRef<HTMLInputElement | null>(null)

  const [name, setName] = useState(me?.name || "")
  const [email, setEmail] = useState(me?.email || "")
  const [info, setInfo] = useState(me?.info || "")
  const [picture, setPicture] = useState<File | null>(null)
  const [loading, setLoading] = useState(false)
  const [success, setSuccess] = useState(false)

  if (!me) return null

  function getShortFileName() {
	if (!me) {
	return <div>Loading...</div>
	}

    if (picture) {
      const name = picture.name
      const ext = name.split(".").pop()
      return name.slice(0, 7) + "..." + ext
    }

    if (!me.picture) return "default.png"

    const original = me.picture.split("/").pop() || "avatar.png"
    const ext = original.split(".").pop()
    return original.slice(0, 7) + "..." + ext
  }

  async function handleSave() {
  if (!me) return

  setLoading(true)
  setSuccess(false)

  const payload: UpdateUserPayload = {}

  if (name !== me.name) payload.name = name
  if (email !== me.email) payload.email = email
  if (info !== me.info) payload.info = info

  if (Object.keys(payload).length > 0) {
    await updateUser(me.login, payload)
  }

  if (picture) {
    await uploadAvatar(me.login, picture)
  }

  setLoading(false)
  setSuccess(true)
}

  async function handleDeleteAvatar() {
  if (!me) return
  await deleteAvatar(me.login)
  setPicture(null)
}

  return (
    <div style={containerStyle}>
      <h2>Profile</h2>

      {/* ===== Avatar ===== */}
      <div style={avatarSection}>
        <div style={avatarWrapper}>
          {me.picture ? (
            <img src={me.picture} style={avatarImg} />
          ) : (
            <div style={avatarPlaceholder}>
              {me.login[0].toUpperCase()}
            </div>
          )}
        </div>

        <div style={fileInfoBlock}>
          <div style={fileNameStyle}>
            {getShortFileName()}
          </div>

          <div style={{ display: "flex", gap: 8 }}>
            <button onClick={() => fileInputRef.current?.click()}>
              Change
            </button>
            <button onClick={handleDeleteAvatar}>
              Delete
            </button>
          </div>

          <input
            ref={fileInputRef}
            type="file"
            accept="image/*"
            style={{ display: "none" }}
            onChange={(e) => {
              if (e.target.files?.[0]) {
                setPicture(e.target.files[0])
              }
            }}
          />
        </div>
      </div>

      {/* ===== Login (readonly) ===== */}
      <div style={field}>
        <label>Login</label>
        <input value={me.login} disabled />
      </div>

      {/* ===== Name ===== */}
      <div style={field}>
        <label>Nickname</label>
        <input value={name} onChange={e => setName(e.target.value)} />
      </div>

      {/* ===== Email ===== */}
      <div style={field}>
        <label>Email</label>
        <input value={email} onChange={e => setEmail(e.target.value)} />
      </div>

      {/* ===== Info ===== */}
      <div style={field}>
        <label>Info</label>
        <textarea
          value={info}
          onChange={e => setInfo(e.target.value)}
          rows={4}
        />
      </div>

      <div style={{ marginTop: 20 }}>
        <button onClick={handleSave} disabled={loading}>
          {loading ? "Saving..." : "Save"}
        </button>
        {success && <span style={{ marginLeft: 10, color: "green" }}>Saved ✓</span>}
      </div>
    </div>
  )
}

const containerStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: 15,
  maxWidth: 400
}

const avatarSection: React.CSSProperties = {
  display: "flex",
  gap: 20,
  alignItems: "center"
}

const avatarWrapper: React.CSSProperties = {
  width: 100,
  height: 100,
  borderRadius: "50%",
  overflow: "hidden",
  background: "#ddd",
  display: "flex",
  alignItems: "center",
  justifyContent: "center"
}

const avatarImg: React.CSSProperties = {
  width: "100%",
  height: "100%",
  objectFit: "cover"
}

const avatarPlaceholder: React.CSSProperties = {
  fontSize: 36,
  fontWeight: 600
}

const fileInfoBlock: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: 8
}

const fileNameStyle: React.CSSProperties = {
  padding: "6px 10px",
  border: "1px solid #ccc",
  borderRadius: 6,
  background: "#f5f5f5",
  fontSize: 13
}

const field: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: 4
}