//sozvon-client\src\components\settings\ProfileSettings.tsx

import { useRef, useState } from "react"
import { useChatContext } from "../../context/ChatContext"
import {
  updateUser,
  uploadAvatar,
  deleteAvatar,
  deleteUser,
  UpdateUserPayload
} from "../../api/users"
import { useNavigate } from "react-router-dom"

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
  const navigate = useNavigate()
  const [showDeleteModal, setShowDeleteModal] = useState(false)
  const [deleting, setDeleting] = useState(false)
  
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

async function handleDeleteUser() {
  if (!me) return

  try {
    setDeleting(true)
    await deleteUser(me.login)
    navigate("/login")
  } catch (err) {
    console.error("Failed to delete user", err)
  } finally {
    setDeleting(false)
  }
}

  return (
	<>
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

      <div style={{ marginTop: 10 }}>
        <button onClick={handleSave} disabled={loading}>
          {loading ? "Saving..." : "Save"}
        </button>
        {success && <span style={{ marginLeft: 10, color: "green" }}>Saved ✓</span>}
      </div>

	  <div style={{ marginTop: 20 }}>
		<button
			onClick={() => setShowDeleteModal(true)}
			style={{
			background: "#ff4d4f",
			color: "white",
			border: "none",
			padding: "8px 14px",
			borderRadius: 6,
			cursor: "pointer"
    }}
  >
    Delete Profile
  </button>
</div>
    </div>
	{showDeleteModal && (
  <div style={modalOverlay}>
    <div style={modalContent}>
      <h3>Approving deletion</h3>
      <p>Are you sure you want to delete your profile? This action is irreversible.</p>

      <div style={{ display: "flex", justifyContent: "flex-end", gap: 10 }}>
        <button onClick={() => setShowDeleteModal(false)}>
          Cancel
        </button>

        <button
          onClick={handleDeleteUser}
          disabled={deleting}
          style={{
            background: "#ff4d4f",
            color: "white",
            border: "none",
            padding: "6px 12px",
            borderRadius: 6
          }}
        >
          {deleting ? "deletion..." : "delete"}
        </button>
      </div>
    </div>
  </div>
)}
</>
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

const modalOverlay: React.CSSProperties = {
  position: "fixed",
  top: 0,
  left: 0,
  right: 0,
  bottom: 0,
  background: "rgba(0,0,0,0.4)",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  zIndex: 1000
}

const modalContent: React.CSSProperties = {
  background: "white",
  padding: 20,
  borderRadius: 10,
  width: 350,
  display: "flex",
  flexDirection: "column",
  gap: 15
}