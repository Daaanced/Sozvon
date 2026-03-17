//sozvon-client\src\components\chat\MessageBubble.tsx

import { styles } from "./chat.styles";
import type { Message } from "./chat.types";
import type { User } from "../../api/users";
import { useState, useRef } from "react";
import { createPortal } from "react-dom";

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

  const menuBtnRef = useRef<HTMLDivElement>(null);
  const [menuPos, setMenuPos] = useState<{ top: number; right: number } | null>(
    null,
  );
  const closeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  function scheduleClose() {
    closeTimerRef.current = setTimeout(() => closeMenu(), 700);
  }

  function cancelClose() {
    if (closeTimerRef.current) clearTimeout(closeTimerRef.current);
  }

  function closeMenu() {
    setMenuOpen(false);
    setHovered(false);
  }

  function handleMenuToggle() {
    if (menuOpen) {
      setMenuOpen(false);
      return;
    }
    if (!menuBtnRef.current) return;

    const rect = menuBtnRef.current.getBoundingClientRect();
    const menuHeight = menuItems.length * 37;
    const spaceBelow = window.innerHeight - rect.bottom;

    setMenuPos({
      top:
        spaceBelow < menuHeight ? rect.top - menuHeight - 4 : rect.bottom + 4,
      right: window.innerWidth - rect.right,
    });

    setMenuOpen(true);
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
        if (e.currentTarget.contains(e.relatedTarget as Node)) return;
        if (!menuOpen) setHovered(false);
      }}
      style={{
        ...styles.groupStartWrapper,
        ...styles.bubbleWrapper,
        ...(highlight && styles.bubbleHighlight),
        ...(hovered && styles.bubbleHover),
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

        {/* Reply */}
        {m.replyToMessage && (
          <div
            onClick={() => onScrollToMessage(m.replyToMessage!.id)}
            style={styles.replyBlock}
          >
            <div style={styles.replyAuthor}>
              {getUser(m.replyToMessage.senderId).name}
            </div>
            <div style={styles.replyText}>
              {m.replyToMessage.text
                ? m.replyToMessage.text.slice(0, 80) +
                  (m.replyToMessage.text.length > 80 ? "..." : "")
                : "📎 вложение"}
            </div>
          </div>
        )}

        {/* Forwarded */}
        {m.forwardedFrom && (
          <div style={styles.forwardBlock}>
            <div style={styles.forwardAuthor}>
              ↪ {getUser(m.forwardedFrom.senderId).name}
            </div>

            {m.forwardedFrom.text && (
              <div style={styles.forwardText}>
                {m.forwardedFrom.text.slice(0, 120)}
                {m.forwardedFrom.text.length > 120 ? "..." : ""}
              </div>
            )}

            {(m.forwardedFrom.attachments?.length ?? 0) > 0 && (
              <div style={styles.forwardAttachments}>
                {m.forwardedFrom.attachments!.map((att) => {
                  if (isImage(att.mimeType)) {
                    return (
                      <img
                        key={att.id}
                        src={att.url}
                        style={styles.forwardImage}
                        onClick={() => onImageClick(att.url)}
                      />
                    );
                  }
                  if (isVideo(att.mimeType)) {
                    return (
                      <video
                        key={att.id}
                        src={att.url}
                        controls
                        style={styles.forwardVideo}
                      />
                    );
                  }
                  return (
                    <a
                      key={att.id}
                      href={att.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      style={styles.forwardFileLink}
                    >
                      📎 {att.fileName}
                    </a>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {/* Text */}
        {m.deletedAt ? (
          <div style={{ ...styles.messageText, ...styles.deletedMessage }}>
            Сообщение удалено
          </div>
        ) : (
          <>
            {m.text && <div style={styles.messageText}>{m.text}</div>}
            {m.editedAt && (
              <span style={styles.editedLabel}>(изменено)</span>
            )}
          </>
        )}

        {/* Images */}
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
                style={multipleImages ? styles.gridImage : styles.singleImage}
                onClick={() => onImageClick(att.url)}
              />
            ))}
          </div>
        )}

        {/* Videos */}
        {videoAttachments.map((att) => (
          <video
            key={att.id}
            src={att.url}
            controls
            style={styles.inlineVideo}
          />
        ))}

        {/* Files */}
        {fileAttachments.length > 0 && (
          <div style={styles.attachments}>
            {fileAttachments.map((att) => (
              <a
                key={att.id}
                href={att.url}
                target="_blank"
                rel="noopener noreferrer"
                style={styles.fileLink}
              >
                <span style={styles.fileIcon}>📎</span>
                <span style={styles.fileName}>{att.fileName}</span>
                <span style={styles.fileSize}>{formatBytes(att.size)}</span>
              </a>
            ))}
          </div>
        )}

        {/* Actions */}
        {hovered && (
          <div style={styles.actionsOverlay}>
            <button onClick={() => onReply(m)} style={styles.actionBtn}>
              ↩
            </button>

            <button onClick={() => onForward(m)} style={styles.actionBtn}>
              ↪
            </button>

            <div
              ref={menuBtnRef}
              onMouseEnter={cancelClose}
              onMouseLeave={scheduleClose}
              style={styles.menuWrapper}
            >
              <button onClick={handleMenuToggle} style={styles.actionBtn}>
                •••
              </button>
            </div>

            {menuOpen &&
              menuPos &&
              createPortal(
                <>
                  <div
                    style={styles.portalOverlay}
                    onClick={closeMenu}
                  />
                  <div
                    style={{
                      ...styles.dropdownMenu,
                      top: menuPos.top,
                      right: menuPos.right,
                    }}
                    onMouseEnter={cancelClose}
                    onMouseLeave={scheduleClose}
                  >
                    {menuItems.map((item) => (
                      <div
                        key={item.label}
                        onClick={item.action}
                        style={{
                          ...styles.dropdownItem,
                          ...(item.danger && styles.dropdownItemDanger),
                        }}
                      >
                        {item.label}
                      </div>
                    ))}
                  </div>
                </>,
                document.body,
              )}
          </div>
        )}
      </div>
    </div>
  );
}