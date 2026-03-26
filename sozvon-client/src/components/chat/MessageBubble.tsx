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
      {isGroupStart ? (
        <img src={user.picture} style={styles.avatar} alt={user.name} />
      ) : (
        <div style={styles.timeColumn}>
          <span style={styles.smallTime}>{formatTime(m.createdAt)}</span>
        </div>
      )}

      <div style={styles.groupContent}>
        {isGroupStart && (
          <div style={styles.headerLine}>
            <span style={styles.userName}>{user.name}</span>
            <span style={styles.fullDate}>{formatFullDate(m.createdAt)}</span>
          </div>
        )}

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

        {m.forwardedFrom && (
          <div style={styles.forwardBlock}>
            <div style={styles.forwardAuthor}>
              ↪ {getUser(m.forwardedFrom.senderId).name}
            </div>

            {m.forwardedFrom.text && (
              <div style={styles.forwardText}>
                {m.forwardedFrom.text.slice(0, 120)}
              </div>
            )}

            {(m.forwardedFrom.attachments?.length ?? 0) > 0 &&
              (() => {
                const fwd = m.forwardedFrom;
                const fwdImages = fwd.attachments!.filter((a) =>
                  isImage(a.mimeType),
                );
                const fwdVideos = fwd.attachments!.filter((a) =>
                  isVideo(a.mimeType),
                );
                const fwdFiles = fwd.attachments!.filter(
                  (a) => !isImage(a.mimeType) && !isVideo(a.mimeType),
                );
                return (
                  <>
                    {fwdImages.length === 1 ? (
                      <div
                        style={{
                          ...styles.singleImageWrapper,
                          marginTop: m.text ? 6 : 0,
                          aspectRatio:
                            fwdImages[0].width && fwdImages[0].height
                              ? `${fwdImages[0].width} / ${fwdImages[0].height}`
                              : "1 / 1",
                          maxWidth: 320,
                          width: "100%",
                        }}
                        onClick={() => onImageClick(fwdImages[0].url)}
                      >
                        <img
                          src={fwdImages[0].url}
                          style={{
                            ...styles.singleImageEl,
                            aspectRatio: undefined,
                          }}
                        />
                        <span style={styles.singleImageTime}>
                          {formatTime(m.createdAt)}
                        </span>
                      </div>
                    ) : fwdImages.length > 1 ? (
                      <div
                        style={{
                          ...styles.imageGrid,
                          gridTemplateColumns: "repeat(auto-fill, 80px)",
                          marginTop: m.forwardedFrom.text ? 6 : 0,
                        }}
                      >
                        {fwdImages.map((att) => (
                          <div
                            key={att.id}
                            style={{
                              width: 80,
                              aspectRatio: "1 / 1",
                              overflow: "hidden",
                              borderRadius: 6,
                            }}
                          >
                            <img
                              src={att.url}
                              style={styles.gridImage}
                              onClick={() => onImageClick(att.url)}
                            />
                          </div>
                        ))}
                      </div>
                    ) : null}

                    {fwdVideos.map((att) => (
                      <div
                        key={att.id}
                        style={{
                          ...styles.singleVideoWrapper,
                          marginTop: m.text ? 6 : 0,
                          aspectRatio:
                            att.width && att.height
                              ? `${att.width} / ${att.height}`
                              : "16 / 9",
                          maxWidth: 320,
                          width: "100%",
                        }}
                      >
                        <video
                          src={att.url}
                          controls
                          style={{
                            ...styles.singleVideoEl,
                            aspectRatio: undefined,
                            width: "100%",
                            height: "100%",
                          }}
                        />
                      </div>
                    ))}

                    {fwdFiles.length > 0 && (
                      <div style={styles.attachments}>
                        {fwdFiles.map((att) => (
                          <a
                            key={att.id}
                            href={att.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            style={styles.fileLink}
                          >
                            <span style={styles.fileIcon}>📎</span>
                            <span style={styles.fileName}>{att.fileName}</span>
                            <span style={styles.fileSize}>
                              {formatBytes(att.size)}
                            </span>
                          </a>
                        ))}
                      </div>
                    )}
                  </>
                );
              })()}
          </div>
        )}

        {m.deletedAt ? (
          <div style={{ ...styles.messageText, ...styles.deletedMessage }}>
            Сообщение удалено
          </div>
        ) : (
          <>
            {m.text && <div style={styles.messageText}>{m.text}</div>}
            {m.editedAt && <span style={styles.editedLabel}>(изменено)</span>}
          </>
        )}

        {imageAttachments.length === 1 ? (
          <div
            style={{
              ...styles.singleImageWrapper,
              marginTop: m.text ? 6 : 0,
              aspectRatio:
                imageAttachments[0].width && imageAttachments[0].height
                  ? `${imageAttachments[0].width} / ${imageAttachments[0].height}`
                  : "1 / 1",
              maxWidth: 320,
              width: "100%",
            }}
            onClick={() => onImageClick(imageAttachments[0].url)}
          >
            <img
              src={imageAttachments[0].url}
              style={{ ...styles.singleImageEl, aspectRatio: undefined }}
            />
            <span style={styles.singleImageTime}>
              {formatTime(m.createdAt)}
            </span>
          </div>
        ) : imageAttachments.length > 1 ? (
          <div
            style={{
              ...styles.imageGrid,
              gridTemplateColumns: "repeat(auto-fill, 80px)",
              marginTop: m.text ? 10 : 4,
              marginBottom: 4,
            }}
          >
            {imageAttachments.map((att) => (
              <div
                key={att.id}
                style={{
                  width: 80,
                  aspectRatio: "1 / 1",
                  overflow: "hidden",
                  borderRadius: 6,
                }}
              >
                <img
                  src={att.url}
                  style={styles.gridImage}
                  onClick={() => onImageClick(att.url)}
                />
              </div>
            ))}
          </div>
        ) : null}

        {videoAttachments.map((att) => (
          <div
            key={att.id}
            style={{
              ...styles.singleVideoWrapper,
              marginTop: m.text ? 6 : 0,
              aspectRatio:
                att.width && att.height
                  ? `${att.width} / ${att.height}`
                  : "16 / 9",
              maxWidth: 320,
              width: "100%",
            }}
          >
            <video
              src={att.url}
              controls
              style={{
                ...styles.singleVideoEl,
                aspectRatio: undefined,
                width: "100%",
                height: "100%",
              }}
            />
          </div>
        ))}

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
                  <div style={styles.portalOverlay} onClick={closeMenu} />
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
