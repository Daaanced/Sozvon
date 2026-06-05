// sozvon-client/src/components/Sidebar.tsx

import { useNavigate, useLocation } from "react-router-dom";
import { useChatContext, DELETED_USER } from "../context/ChatContext";
import { useState } from "react";
import SettingsModal from "./SettingsModal";
import CreateGroupModal from "./CreateGroupModal";
import { useVoiceContext } from "../context/VoiceContext";

export default function Sidebar() {
  const { chats, myId, myLogin, me, getSafeUser, unread } = useChatContext();
  //   console.log(
  //     "[Sidebar] render, chats order:",
  //     chats.map((c) => `${c.chatId.slice(0, 8)}=${c.updatedAt?.slice(11, 19)}`),
  //   );
  const navigate = useNavigate();
  const location = useLocation();
  const [open, setOpen] = useState(false);
  const [groupModalOpen, setGroupModalOpen] = useState(false);
  const { activeRoomName, leaveRoom } = useVoiceContext();

  const isRooms = location.pathname === "/app/rooms";

  return (
    <div style={styles.sidebar}>
      {/* ВЕРХНИЕ КОЛОНКИ */}
      <div style={styles.columns}>
        {/* SERVER LIST */}
        <div style={styles.serverBar}>
          <div
            style={styles.serverBtn}
            onClick={() => navigate("/app")}
            title="Direct Messages"
          >
            💬
          </div>

          <div style={styles.divider} />

          <div
            style={{ ...styles.serverBtn, ...styles.addBtn }}
            onClick={() => console.log("create server")}
            title="Create Server"
          >
            +
          </div>
        </div>

        {/* CHAT SIDEBAR */}
        <div style={styles.chatSidebar}>
          <div
            style={{
              ...styles.sectionBtn,
              background: isRooms ? "#e0e0ff" : "#fff",
            }}
            onClick={() => navigate("/app/rooms")}
          >
            🔊 Voice rooms
          </div>
          <div style={styles.sectionBtn} onClick={() => navigate("/app")}>
            🔍 Search users
          </div>
          <div style={styles.dmHeader}>
            <span style={styles.dmLabel}>Direct messages</span>
            <button
              style={styles.dmAddBtn}
              onClick={() => setGroupModalOpen(true)}
            >
              +
            </button>
          </div>
          {groupModalOpen && (
            <CreateGroupModal onClose={() => setGroupModalOpen(false)} />
          )}
          <div style={styles.chatList}>
            {chats.map((chat) => {
              const isActive = location.pathname.endsWith(chat.chatId);
              const hasUnread = unread[chat.chatId] && !isActive;
              console.log(
                "[Sidebar] chat:",
                chat.chatId.slice(0, 8),
                "type:",
                chat.type,
                "name:",
                chat.name,
              );

              const isGroup = chat.type === "group";
              const displayName = isGroup
                ? (chat.name ?? "Групповой чат")
                : (() => {
                    const withId = chat.members.find((m) => m !== myId);
                    return withId
                      ? getSafeUser(withId).name
                      : DELETED_USER.name;
                  })();
              const displayPicture = isGroup
                ? "https://zvonya.ru/api/static/avatars/group_default.png"
                : (() => {
                    const withId = chat.members.find((m) => m !== myId);
                    return withId
                      ? getSafeUser(withId).picture
                      : DELETED_USER.picture;
                  })();

              return (
                <div
                  key={chat.chatId}
                  onClick={() => navigate(`/app/chats/${chat.chatId}`)}
                  style={{
                    ...styles.chatItem,
                    background: isActive ? "#e0e0ff" : "#fff",
                  }}
                >
                  <div style={styles.avatar}>
                    {displayPicture ? (
                      <img src={displayPicture} style={styles.avatarImg} />
                    ) : (
                      <span>{displayName[0].toUpperCase()}</span>
                    )}
                  </div>

                  <span style={{ flex: 1 }}>{displayName}</span>

                  {hasUnread && <div style={styles.unreadDot} />}
                </div>
              );
            })}
          </div>
        </div>
      </div>

      {activeRoomName && (
        <div style={styles.voicePanel}>
          <div>
            <div style={styles.voiceTitle}>🔊 Voice connected</div>

            <div style={styles.voiceRoom}>{activeRoomName}</div>
          </div>

          <div style={styles.voiceButtons}>
            <button style={styles.voiceBtn} title="Share screen">
              🖥️
            </button>

            <button
              style={styles.voiceBtnLeave}
              onClick={leaveRoom}
              title="Leave"
            >
              📵
            </button>
          </div>
        </div>
      )}
      {/* USER BLOCK */}
      <div style={styles.meBlock}>
        <div style={styles.meInfo}>
          <div style={styles.avatar}>
            {me?.picture ? (
              <img src={me.picture} style={styles.avatarImg} />
            ) : (
              <span>{myLogin[0].toUpperCase()}</span>
            )}
          </div>

          <span>{me?.name || myLogin}</span>
        </div>

        <div style={styles.meButtons}>
          <button style={styles.iconBtn}>🎙️</button>
          <button style={styles.iconBtn}>🎧</button>
          <button style={styles.iconBtn} onClick={() => setOpen(true)}>
            ⚙️
          </button>
          {open && <SettingsModal onClose={() => setOpen(false)} />}
        </div>
      </div>
    </div>
  );
}

const styles = {
  // Внешняя обёртка — горизонтальный flex
  wrapper: {
    display: "flex",
    height: "100vh",
    borderRight: "1px solid #ddd",
  },

  columns: {
    display: "flex",
    flex: 1,
    overflow: "hidden",
  },

  serverBar: {
    width: 56,
    background: "#e3e5e8",
    display: "flex",
    flexDirection: "column" as const,
    alignItems: "center",
    padding: "8px 0",
    gap: 6,
  },

  chatSidebar: {
    flex: 1,
    display: "flex",
    flexDirection: "column" as const,
    padding: 8,
    background: "#f9f9f9",
    minWidth: 0,
  },
  serverBtn: {
    width: 40,
    height: 40,
    borderRadius: "50%",
    background: "#fff",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    cursor: "pointer",
    fontSize: 18,
    flexShrink: 0,
    overflow: "hidden" as const,
    userSelect: "none" as const,
  },
  addBtn: {
    background: "#f0f0f0",
    color: "#4a90e2",
    fontSize: 24,
    fontWeight: 300,
  },
  divider: {
    width: 32,
    height: 2,
    background: "#ccc",
    borderRadius: 1,
  },
  // Основной сайдбар
  sidebar: {
    width: "100%",
    height: "100vh",
    display: "flex",
    flexDirection: "column" as const,
    background: "#f9f9f9",
    padding: 8,
    boxSizing: "border-box" as const,
  },
  sectionBtn: {
    padding: "8px 10px",
    borderRadius: 6,
    background: "#fff",
    cursor: "pointer",
    marginBottom: 4,
    textAlign: "left" as const,
    fontWeight: 500,
    fontSize: 14,
  },
  dmHeader: {
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    padding: "6px 10px",
    marginBottom: 4,
  },
  dmLabel: {
    fontWeight: 600,
    fontSize: 12,
    color: "#555",
    textTransform: "uppercase" as const,
    letterSpacing: "0.05em",
  },
  dmAddBtn: {
    width: 20,
    height: 20,
    background: "none",
    border: "none",
    cursor: "pointer",
    fontSize: 18,
    color: "#555",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    borderRadius: 4,
    padding: 0,
    lineHeight: 1,
  },
  chatList: {
    flex: 1,
    display: "flex",
    flexDirection: "column" as const,
    gap: 2,
    overflowY: "auto" as const,
  },
  chatItem: {
    display: "flex",
    alignItems: "center",
    gap: 10,
    padding: "6px 8px",
    borderRadius: 6,
    cursor: "pointer",
  },
  avatar: {
    width: 36,
    height: 36,
    borderRadius: "50%",
    background: "#ddd",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    overflow: "hidden",
    flexShrink: 0,
  },
  avatarImg: { width: "100%", height: "100%", objectFit: "cover" as const },
  unreadDot: {
    width: 10,
    height: 10,
    borderRadius: "50%",
    background: "#4a90e2",
    flexShrink: 0,
  },
  meBlock: {
    borderTop: "1px solid #ddd",
    paddingTop: 8,
    marginTop: 8,
  },
  meInfo: {
    display: "flex",
    alignItems: "center",
    gap: 10,
    marginBottom: 6,
  },
  meButtons: { display: "flex", justifyContent: "space-between" },
  iconBtn: {
    flex: 1,
    margin: 2,
    padding: 6,
    borderRadius: 6,
    border: "none",
    cursor: "pointer",
  },
  voicePanel: {
    borderTop: "1px solid #ddd",
    padding: "10px",
    background: "#f3f4f6",
    borderRadius: 8,
    marginTop: 8,
  },

  voiceTitle: {
    fontSize: 12,
    color: "#666",
  },

  voiceRoom: {
    fontWeight: 600,
    marginTop: 4,
  },

  voiceButtons: {
    display: "flex",
    gap: 6,
    marginTop: 8,
  },

  voiceBtn: {
    flex: 1,
    padding: 6,
    border: "none",
    borderRadius: 6,
    cursor: "pointer",
  },

  voiceBtnLeave: {
    flex: 1,
    padding: 6,
    border: "none",
    borderRadius: 6,
    cursor: "pointer",
    background: "#dc2626",
    color: "#fff",
  },
};
