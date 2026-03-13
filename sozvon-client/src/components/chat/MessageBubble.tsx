//sozvon-client\src\components\chat\MessageBubble.tsx

import { styles } from "./chat.styles";
import type { Message } from "./chat.types";
import type { User } from "../../api/users";
import { useState } from "react";

type Props = {
  message: Message;
  user: User;
  isGroupStart: boolean;
  onImageClick: (url: string) => void;
  onReply: (message: Message) => void;
  onForward: (message: Message) => void;
  onEdit: (message: Message) => void;
  onDelete: (message: Message) => void;
  myId: number;
  onScrollToMessage: (id: string) => void;
  highlight: boolean;
  setRef: (el: HTMLDivElement | null) => void;
  getUser: (id: number) => User;
};

const btnStyle: React.CSSProperties = {
  background: "rgba(148,144,144,0.92)",
  border: "none",
  borderRadius: 8,
  cursor: "pointer",
  padding: "2px 8px",
  fontSize: 16,
  lineHeight: "22px",
};

function isImage(mimeType: string) {
  return mimeType.startsWith("image/");
}
function isVideo(mimeType: string) {
  return mimeType.startsWith("video/");
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return bytes + " B";
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
  return (bytes / (1024 * 1024)).toFixed(1) + " MB";
}

function formatTime(dateString: string) {
  return new Date(dateString).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatFullDate(dateString: string) {
  const d = new Date(dateString);
  return (
    d.toLocaleDateString("ru-RU") +
    ", " +
    d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
  );
}

export default function MessageBubble({
  message: m,
  user,
  isGroupStart,
  onImageClick,
  onReply,
  onForward,
  onEdit,
  onDelete,
  onScrollToMessage,
  highlight,
  setRef,
  myId,
  getUser,
}: Props) {
  const [hovered, setHovered] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);

  const isOwn = m.senderId === myId;
  const imageAttachments =
    m.attachments?.filter((a) => isImage(a.mimeType)) ?? [];
  const videoAttachments =
    m.attachments?.filter((a) => isVideo(a.mimeType)) ?? [];
  const fileAttachments =
    m.attachments?.filter(
      (a) => !isImage(a.mimeType) && !isVideo(a.mimeType),
    ) ?? [];
  const multipleImages = imageAttachments.length > 1;

  function closeMenu() {
    setMenuOpen(false);
    setHovered(false);
  }

  const menuItems = [
    {
      label: "↩ Ответить",
      action: () => {
        onReply(m);
        closeMenu();
      },
    },
    {
      label: "↪ Переслать",
      action: () => {
        onForward(m);
        closeMenu();
      },
    },
    ...(isOwn && !m.deletedAt
      ? [
          {
            label: "✏️ Изменить",
            action: () => {
              onEdit(m);
              closeMenu();
            },
          },
          {
            label: "🗑 Удалить",
            action: () => {
              onDelete(m);
              closeMenu();
            },
            danger: true,
          },
        ]
      : []),
  ];

  return (
    <div
      ref={setRef}
      data-bubble="true"
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={(e) => {
        // Не скрывать если мышь ушла на дочерний элемент (напр. меню)
        if (e.currentTarget.contains(e.relatedTarget as Node)) return;
        if (!menuOpen) setHovered(false);
      }}
      style={{
        ...styles.groupStartWrapper,
        background: highlight ? "#c8f6f0" : hovered ? "#c9daed" : undefined,
        transition: "background 0.2s",
        position: "relative",
      }}
    >
      {/* Левая колонка */}
      {isGroupStart ? (
        <img src={user.picture} style={styles.avatar} alt={user.name} />
      ) : (
        <div style={styles.timeColumn}>
          <span style={styles.smallTime}>{formatTime(m.createdAt)}</span>
        </div>
      )}

      {/* Правая колонка */}
      <div style={styles.groupContent}>
        {isGroupStart && (
          <div style={styles.headerLine}>
            <span style={styles.userName}>{user.name}</span>
            <span style={styles.fullDate}>{formatFullDate(m.createdAt)}</span>
          </div>
        )}

        {/* Ответ */}
        {m.replyToMessage && (
          <div
            onClick={() => onScrollToMessage(m.replyToMessage!.id)}
            style={{
              borderLeft: "3px solid #888",
              background: "rgba(180,180,180,0.12)",
              borderRadius: "0 6px 6px 0",
              padding: "6px 8px",
              marginBottom: 4,
              fontSize: 13,
              cursor: "pointer",
            }}
          >
            <div style={{ fontWeight: 600, marginBottom: 2, opacity: 0.8 }}>
              {getUser(m.replyToMessage.senderId).name}
            </div>
            <div style={{ opacity: 0.7 }}>
              {m.replyToMessage.text
                ? m.replyToMessage.text.slice(0, 80) +
                  (m.replyToMessage.text.length > 80 ? "..." : "")
                : "📎 вложение"}
            </div>
          </div>
        )}

        {/* Пересланное */}
        {m.forwardedFrom && (
          <div
            style={{
              borderLeft: "3px solid #A4C7F0",
              background: "rgba(164,199,240,0.15)",
              borderRadius: "0 6px 6px 0",
              padding: "6px 8px",
              marginBottom: 4,
              fontSize: 13,
            }}
          >
            <div style={{ fontWeight: 600, marginBottom: 2, opacity: 0.8 }}>
              ↪ {getUser(m.forwardedFrom.senderId).name}
            </div>
            {m.forwardedFrom.text && (
              <div style={{ opacity: 0.7 }}>
                {m.forwardedFrom.text.slice(0, 120)}
                {m.forwardedFrom.text.length > 120 ? "..." : ""}
              </div>
            )}
            {(m.forwardedFrom.attachments?.length ?? 0) > 0 && (
              <div
                style={{
                  marginTop: 4,
                  display: "flex",
                  flexWrap: "wrap",
                  gap: 4,
                }}
              >
                {m.forwardedFrom.attachments!.map((att) => {
                  if (att.mimeType.startsWith("image/")) {
                    return (
                      <img
                        key={att.id}
                        src={att.url}
                        alt={att.fileName}
                        style={{
                          maxWidth: 120,
                          maxHeight: 80,
                          borderRadius: 4,
                          cursor: "pointer",
                          objectFit: "cover",
                        }}
                        onClick={() => onImageClick(att.url)}
                      />
                    );
                  }
                  if (att.mimeType.startsWith("video/")) {
                    return (
                      <video
                        key={att.id}
                        src={att.url}
                        controls
                        style={{ maxWidth: 200, borderRadius: 4 }}
                      />
                    );
                  }
                  return (
                    <a
                      key={att.id}
                      href={att.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      style={{
                        display: "flex",
                        alignItems: "center",
                        gap: 4,
                        fontSize: 12,
                        opacity: 0.8,
                      }}
                    >
                      <span>📎</span>
                      <span>{att.fileName}</span>
                    </a>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {/* Текст / удалено */}
        {m.deletedAt ? (
          <div
            style={{ ...styles.messageText, opacity: 0.5, fontStyle: "italic" }}
          >
            Сообщение удалено
          </div>
        ) : (
          <>
            {m.text && <div style={styles.messageText}>{m.text}</div>}
            {m.editedAt && (
              <span style={{ fontSize: 11, opacity: 0.5 }}>(изменено)</span>
            )}
          </>
        )}

        {/* Изображения */}
        {imageAttachments.length > 0 && (
          <div
            style={{
              ...styles.imageGrid,
              gridTemplateColumns: multipleImages
                ? "repeat(auto-fill, 80px)"
                : "1fr",
            }}
          >
            {imageAttachments.map((att) => (
              <img
                key={att.id}
                src={att.url}
                alt={att.fileName}
                title={att.fileName}
                style={multipleImages ? styles.gridImage : styles.singleImage}
                onClick={() => onImageClick(att.url)}
              />
            ))}
          </div>
        )}

        {/* Видео */}
        {videoAttachments.map((att) => (
          <video
            key={att.id}
            src={att.url}
            controls
            style={styles.inlineVideo}
            title={att.fileName}
          />
        ))}

        {/* Файлы */}
        {fileAttachments.length > 0 && (
          <div style={styles.attachments}>
            {fileAttachments.map((att) => (
              <div key={att.id}>
                <a
                  href={att.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  style={styles.fileLink}
                >
                  <span style={styles.fileIcon}>📎</span>
                  <span style={styles.fileName}>{att.fileName}</span>
                  <span style={styles.fileSize}>{formatBytes(att.size)}</span>
                </a>
              </div>
            ))}
          </div>
        )}

        {/* Кнопки действий */}
        {hovered && (
          <div
            style={{
              position: "absolute",
              top: 4,
              right: 8,
              zIndex: 10,
              display: "flex",
              alignItems: "center",
              gap: 4,
            }}
          >
            <button
              onClick={() => onReply(m)}
              title="Ответить"
              style={btnStyle}
            >
              ↩
            </button>
            <button
              onClick={() => onForward(m)}
              title="Переслать"
              style={btnStyle}
            >
              ↪
            </button>

            {/* Меню ••• */}
            <div style={{ position: "relative" }}>
              <button onClick={() => setMenuOpen((v) => !v)} style={btnStyle}>
                •••
              </button>

              {menuOpen && (
                <div
                  onMouseEnter={() => setHovered(true)}
                  onMouseLeave={(e) => {
                    // Закрываем только если мышь ушла за пределы пузыря
                    const bubble = e.currentTarget.closest("[data-bubble]");
                    if (bubble && bubble.contains(e.relatedTarget as Node))
                      return;
                    closeMenu();
                  }}
                  style={{
                    position: "absolute",
                    right: 0,
                    top: 30,
                    background: "#fff",
                    borderRadius: 8,
                    boxShadow: "0 2px 12px rgba(0,0,0,0.15)",
                    minWidth: 160,
                    zIndex: 20,
                    overflow: "hidden",
                  }}
                >
                  {menuItems.map((item) => (
                    <div
                      key={item.label}
                      onClick={item.action}
                      style={{
                        padding: "8px 16px",
                        cursor: "pointer",
                        fontSize: 14,
                        color: (item as any).danger ? "#e53935" : "#222",
                      }}
                      onMouseEnter={(e) =>
                        (e.currentTarget.style.background = "#f5f5f5")
                      }
                      onMouseLeave={(e) =>
                        (e.currentTarget.style.background = "transparent")
                      }
                    >
                      {item.label}
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
