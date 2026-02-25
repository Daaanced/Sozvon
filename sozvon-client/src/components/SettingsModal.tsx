// sozvon-client/src/components/SettingsModal.tsx
import { useState } from "react"
import ProfileSettings from "./settings/ProfileSettings"
import AppearanceSettings from "./settings/AppearanceSettings"
import SoundSettings from "./settings/SoundSettings"

type Props = {
  onClose: () => void
}

type Section = "profile" | "appearance" | "sound"

export default function SettingsModal({ onClose }: Props) {
  const [active, setActive] = useState<Section>("profile")

  function renderContent() {
    switch (active) {
      case "profile":
        return <ProfileSettings />
      case "appearance":
        return <AppearanceSettings />
      case "sound":
        return <SoundSettings />
      default:
        return null
    }
  }

  return (
    <div style={overlayStyle} onClick={onClose}>
      <div
  		style={modalStyle}
  		onClick={(e) => e.stopPropagation()}
	  >
        <div style={layoutStyle}>
          {/* LEFT MENU */}
          <div style={menuStyle}>
            <MenuItem
              label="Profile"
              active={active === "profile"}
              onClick={() => setActive("profile")}
            />
            <MenuItem
              label="Appearance"
              active={active === "appearance"}
              onClick={() => setActive("appearance")}
            />
            <MenuItem
              label="Sound"
              active={active === "sound"}
              onClick={() => setActive("sound")}
            />
            <div style={{ flex: 1 }} />
            <button style={logoutStyle}>Logout</button>
          </div>

          {/* RIGHT CONTENT */}
          <div style={contentStyle}>
            {renderContent()}
          </div>
        </div>
      </div>
    </div>
  )
}

function MenuItem({ label, active, onClick }: any) {
  return (
    <div
      onClick={onClick}
      style={{
        padding: "10px 12px",
        borderRadius: 6,
        cursor: "pointer",
        background: active ? "#e0e0ff" : "transparent",
        fontWeight: active ? 600 : 400
      }}
    >
      {label}
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
  width: 700,
  height: 500,
  borderRadius: 10,
  overflow: "hidden"
}

const layoutStyle: React.CSSProperties = {
  display: "flex",
  height: "100%"
}

const menuStyle: React.CSSProperties = {
  width: 180,
  borderRight: "1px solid #ddd",
  padding: 10,
  display: "flex",
  flexDirection: "column",
  gap: 6
}

const contentStyle: React.CSSProperties = {
  flex: 1,
  padding: 20,
  overflowY: "auto"
}

const logoutStyle: React.CSSProperties = {
  padding: 8,
  borderRadius: 6,
  border: "none",
  cursor: "pointer",
  background: "#ffe0e0"
}
