// sozvon-client/src/components/UserInfo.tsx
import { useLocation } from "react-router-dom";
import { useChatContext, DELETED_USER } from "../context/ChatContext";

export default function UserInfo() {
  const { chats, myId, getSafeUser } = useChatContext();
  const location = useLocation();

  const chatId = location.pathname.split("/").pop();
  const chat = chats.find((c) => c.chatId === chatId);

  if (!chat) {
    return <div style={{ padding: 16, color: "#aaa" }}>Выберите чат</div>;
  }

  // ── GROUP ──────────────────────────────────────────────
  if (chat.type === "group") {
    return (
      <div style={{ padding: 16 }}>
        <div style={gs.groupName}>{chat.name ?? "Групповой чат"}</div>
        <div style={gs.membersLabel}>Участники ({chat.members.length})</div>
        <div style={gs.memberList}>
          {chat.members.map((id) => {
            const user = getSafeUser(id);
            return (
              <div key={id} style={gs.memberItem}>
                <div style={gs.avatar}>
                  {user.picture ? (
                    <img src={user.picture} style={gs.avatarImg} />
                  ) : (
                    <span style={gs.avatarLetter}>
                      {user.name[0].toUpperCase()}
                    </span>
                  )}
                </div>
                <div>
                  <div style={gs.memberName}>{user.name}</div>
                  {id === myId && <div style={gs.youBadge}>вы</div>}
                </div>
              </div>
            );
          })}
        </div>
      </div>
    );
  }

  // ── DIRECT ─────────────────────────────────────────────
  const withId = chat.members.find((m) => m !== myId);
  const user = withId ? getSafeUser(withId) : DELETED_USER;

  return (
    <div style={{ padding: 16 }}>
      <div style={{ marginBottom: 12 }}>
        {user.picture ? (
          <img
            src={user.picture}
            style={{
              width: 100,
              height: 100,
              borderRadius: "50%",
              objectFit: "cover",
            }}
          />
        ) : (
          <div
            style={{
              width: 100,
              height: 100,
              borderRadius: "50%",
              background: "#dde3f0",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              fontSize: 36,
              fontWeight: 700,
              color: "#4a90e2",
            }}
          >
            {user.login[0].toUpperCase()}
          </div>
        )}
      </div>
      <div>
        <b>Login:</b> {user.login}
      </div>
      <div>
        <b>Name:</b> {user.name}
      </div>
      <div>
        <b>Email:</b> {user.email || "-"}
      </div>
      <div>
        <b>Info:</b> {user.info || "-"}
      </div>
      <div>
        <b>Created:</b> {new Date(user.created_at).toLocaleString()}
      </div>
    </div>
  );
}

const gs: Record<string, React.CSSProperties> = {
  groupName: {
    fontWeight: 700,
    fontSize: 17,
    color: "#222",
    marginBottom: 16,
    wordBreak: "break-word",
  },
  membersLabel: {
    fontSize: 11,
    fontWeight: 600,
    color: "#888",
    textTransform: "uppercase",
    letterSpacing: "0.07em",
    marginBottom: 10,
  },
  memberList: {
    display: "flex",
    flexDirection: "column",
    gap: 8,
  },
  memberItem: {
    display: "flex",
    alignItems: "center",
    gap: 10,
  },
  avatar: {
    width: 38,
    height: 38,
    borderRadius: "50%",
    background: "#dde3f0",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    overflow: "hidden",
    flexShrink: 0,
  },
  avatarImg: { width: "100%", height: "100%", objectFit: "cover" },
  avatarLetter: { fontWeight: 700, fontSize: 16, color: "#4a90e2" },
  memberName: { fontSize: 14, color: "#222", fontWeight: 500 },
  youBadge: {
    fontSize: 11,
    color: "#4a90e2",
    fontWeight: 600,
    marginTop: 1,
  },
};
