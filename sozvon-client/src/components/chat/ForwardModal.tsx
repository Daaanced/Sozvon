// sozvon-client/src/components/chat/ForwardModal.tsx
import { useState } from "react";
import { useChatContext, DELETED_USER } from "../../context/ChatContext";
import { forwardMessages } from "../../api/chats";
import type { Message } from "./chat.types";

type Props = {
  message: Message;
  onClose: () => void;
  onDone: () => void;
};

export default function ForwardModal({ message, onClose, onDone }: Props) {
  const { chats, myId, getSafeUser } = useChatContext();
  const [selectedChatId, setSelectedChatId] = useState<string | null>(null);
  const [comment, setComment] = useState("");
  const [sending, setSending] = useState(false);

  async function handleForward() {
    if (!selectedChatId) return;
    setSending(true);
    try {
      await forwardMessages(
        [message.id],
        selectedChatId,
        comment.trim() || undefined,
      );
      onDone();
      onClose();
    } catch (e) {
      console.error("Forward failed:", e);
    } finally {
      setSending(false);
    }
  }

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0,0,0,0.4)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 100,
      }}
    >
      <div
        style={{
          background: "#fff",
          borderRadius: 12,
          padding: 24,
          width: 360,
          maxHeight: "70vh",
          display: "flex",
          flexDirection: "column",
          gap: 12,
        }}
      >
        <div style={{ fontWeight: 600, fontSize: 16 }}>Переслать сообщение</div>

        <div
          style={{
            flex: 1,
            overflowY: "auto",
            display: "flex",
            flexDirection: "column",
            gap: 4,
          }}
        >
          {chats.map((chat) => {
            const withId = chat.members.find((m) => m !== myId);
            const user = withId ? getSafeUser(withId) : DELETED_USER;
            const isSelected = selectedChatId === chat.chatId;
            return (
              <div
                key={chat.chatId}
                onClick={() => setSelectedChatId(chat.chatId)}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: 10,
                  padding: "8px 10px",
                  borderRadius: 8,
                  cursor: "pointer",
                  background: isSelected ? "#e0e8ff" : "#f5f5f5",
                }}
              >
                <img
                  src={user.picture}
                  style={{ width: 32, height: 32, borderRadius: "50%" }}
                  alt=""
                />
                <span>{user.name}</span>
              </div>
            );
          })}
        </div>

        <input
          value={comment}
          onChange={(e) => setComment(e.target.value)}
          placeholder="Добавить комментарий..."
          style={{
            padding: "8px 10px",
            borderRadius: 8,
            border: "1px solid #ddd",
            fontSize: 14,
          }}
        />

        <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
          <button
            onClick={onClose}
            style={{
              padding: "6px 16px",
              borderRadius: 8,
              border: "1px solid #ddd",
              cursor: "pointer",
              background: "#fff",
            }}
          >
            Отмена
          </button>
          <button
            onClick={handleForward}
            disabled={!selectedChatId || sending}
            style={{
              padding: "6px 16px",
              borderRadius: 8,
              border: "none",
              cursor: "pointer",
              background: "#4a90e2",
              color: "#fff",
              opacity: !selectedChatId || sending ? 0.5 : 1,
            }}
          >
            {sending ? "..." : "Переслать"}
          </button>
        </div>
      </div>
    </div>
  );
}
