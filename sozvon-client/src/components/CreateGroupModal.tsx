// sozvon-client/src/components/CreateGroupModal.tsx
import { useState } from "react";
import { useChatContext } from "../context/ChatContext";
import { createGroupChat } from "../api/chats";

interface Props {
  onClose: () => void;
  onCreated?: () => void;
}

export default function CreateGroupModal({ onClose, onCreated }: Props) {
  const { chats, myId, getSafeUser } = useChatContext();
  const [name, setName] = useState("");
  const [selected, setSelected] = useState<number[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  // Только direct-чаты — получаем уникальных собеседников
  const directPartners: number[] = [];
  const seen = new Set<number>();
  for (const chat of chats) {
    if (chat.type !== "direct") continue;
    const partnerId = chat.members.find((m) => m !== myId);
    if (partnerId && !seen.has(partnerId)) {
      seen.add(partnerId);
      directPartners.push(partnerId);
    }
  }

  function toggle(id: number) {
    setSelected((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id],
    );
    setError("");
  }

  async function handleCreate() {
    if (!name.trim()) {
      setError("Введите название группы");
      return;
    }
    if (selected.length < 2) {
      setError("Выберите минимум 2 участников");
      return;
    }
    if (selected.length > 9) {
      setError("Максимум 9 участников");
      return;
    }

    setLoading(true);
    setError("");
    try {
      await createGroupChat(name.trim(), selected);
      onCreated?.();
      onClose();
    } catch (e: any) {
      setError(e?.message ?? "Ошибка при создании чата");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={s.overlay} onClick={onClose}>
      <div style={s.modal} onClick={(e) => e.stopPropagation()}>
        <div style={s.header}>
          <span style={s.title}>Создать групповой чат</span>
          <button style={s.closeBtn} onClick={onClose}>
            ✕
          </button>
        </div>

        {/* Название */}
        <label style={s.label}>Название группы *</label>
        <input
          style={s.input}
          placeholder="Введите название..."
          value={name}
          onChange={(e) => {
            setName(e.target.value);
            setError("");
          }}
          maxLength={64}
        />

        {/* Список участников */}
        <label style={s.label}>
          Участники ({selected.length}/9, минимум 2)
        </label>

        {directPartners.length === 0 ? (
          <div style={s.empty}>Нет доступных пользователей</div>
        ) : (
          <div style={s.list}>
            {directPartners.map((id) => {
              const user = getSafeUser(id);
              const isSelected = selected.includes(id);
              return (
                <div
                  key={id}
                  style={{
                    ...s.userItem,
                    background: isSelected ? "#e8eeff" : "#fff",
                    border: isSelected
                      ? "1.5px solid #4a90e2"
                      : "1.5px solid #eee",
                  }}
                  onClick={() => {
                    if (!isSelected && selected.length >= 9) {
                      setError("Максимум 9 участников");
                      return;
                    }
                    toggle(id);
                  }}
                >
                  <div style={s.avatar}>
                    {user.picture ? (
                      <img src={user.picture} style={s.avatarImg} />
                    ) : (
                      <span style={s.avatarLetter}>
                        {user.name[0].toUpperCase()}
                      </span>
                    )}
                  </div>
                  <span style={s.userName}>{user.name}</span>
                  {isSelected && <span style={s.check}>✓</span>}
                </div>
              );
            })}
          </div>
        )}

        {error && <div style={s.error}>{error}</div>}

        <div style={s.footer}>
          <button style={s.cancelBtn} onClick={onClose} disabled={loading}>
            Отмена
          </button>
          <button
            style={{
              ...s.createBtn,
              opacity: loading ? 0.7 : 1,
            }}
            onClick={handleCreate}
            disabled={loading}
          >
            {loading ? "Создание..." : "Создать"}
          </button>
        </div>
      </div>
    </div>
  );
}

const s: Record<string, React.CSSProperties> = {
  overlay: {
    position: "fixed",
    inset: 0,
    background: "rgba(0,0,0,0.35)",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    zIndex: 1000,
  },
  modal: {
    background: "#fff",
    borderRadius: 12,
    padding: 24,
    width: 380,
    maxWidth: "calc(100vw - 32px)",
    boxShadow: "0 8px 32px rgba(0,0,0,0.18)",
    display: "flex",
    flexDirection: "column",
    gap: 10,
  },
  header: {
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    marginBottom: 4,
  },
  title: {
    fontWeight: 700,
    fontSize: 16,
    color: "#222",
  },
  closeBtn: {
    background: "none",
    border: "none",
    cursor: "pointer",
    fontSize: 16,
    color: "#888",
    padding: 4,
    borderRadius: 6,
    lineHeight: 1,
  },
  label: {
    fontSize: 12,
    fontWeight: 600,
    color: "#555",
    textTransform: "uppercase",
    letterSpacing: "0.05em",
    marginBottom: 2,
  },
  input: {
    width: "100%",
    padding: "9px 12px",
    borderRadius: 8,
    border: "1.5px solid #ddd",
    fontSize: 14,
    outline: "none",
    boxSizing: "border-box",
    transition: "border-color 0.15s",
  },
  list: {
    display: "flex",
    flexDirection: "column",
    gap: 6,
    maxHeight: 260,
    overflowY: "auto",
    padding: "2px 0",
  },
  empty: {
    color: "#aaa",
    fontSize: 13,
    textAlign: "center",
    padding: "16px 0",
  },
  userItem: {
    display: "flex",
    alignItems: "center",
    gap: 10,
    padding: "8px 10px",
    borderRadius: 8,
    cursor: "pointer",
    transition: "background 0.1s, border-color 0.1s",
    userSelect: "none",
  },
  avatar: {
    width: 34,
    height: 34,
    borderRadius: "50%",
    background: "#dde3f0",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    overflow: "hidden",
    flexShrink: 0,
  },
  avatarImg: { width: "100%", height: "100%", objectFit: "cover" },
  avatarLetter: { fontWeight: 700, fontSize: 15, color: "#4a90e2" },
  userName: { flex: 1, fontSize: 14, color: "#222" },
  check: { color: "#4a90e2", fontWeight: 700, fontSize: 16 },
  error: {
    color: "#e05555",
    fontSize: 13,
    padding: "4px 2px",
  },
  footer: {
    display: "flex",
    gap: 10,
    marginTop: 6,
    justifyContent: "flex-end",
  },
  cancelBtn: {
    padding: "9px 18px",
    borderRadius: 8,
    border: "1.5px solid #ddd",
    background: "#fff",
    cursor: "pointer",
    fontSize: 14,
    color: "#555",
  },
  createBtn: {
    padding: "9px 22px",
    borderRadius: 8,
    border: "none",
    background: "#4a90e2",
    color: "#fff",
    cursor: "pointer",
    fontSize: 14,
    fontWeight: 600,
    transition: "opacity 0.15s",
  },
};
