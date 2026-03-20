//sozvon-client\src\components\chat\chat.styles.ts
import type React from "react";

export const styles: Record<string, React.CSSProperties> = {
  // Обёртка чата
  chatWrapper: {
    display: "flex",
    flexDirection: "column",
    height: "100%",
    width: "100%",
    position: "relative",
  },

  // Список сообщений
  messageList: {
    flex: 1,
    overflowY: "auto",
    border: "1px solid #ddd",
    padding: 12,
    marginBottom: 8,
  },

  // Пузырь
  groupStartWrapper: {
    display: "flex",
    gap: 12,
    marginTop: 2,
  },
  avatar: {
    width: 40,
    height: 40,
    borderRadius: "50%",
    objectFit: "cover",
    flexShrink: 0,
  },
  groupContent: {
    flex: 1,
    display: "flex",
    flexDirection: "column",
  },
  headerLine: {
    display: "flex",
    alignItems: "baseline",
    gap: 8,
  },
  userName: {
    fontWeight: 600,
    fontSize: 15,
  },
  fullDate: {
    fontSize: 12,
    color: "#888",
  },
  messageText: {
    marginTop: 4,
    lineHeight: 1.4,
    wordBreak: "break-word",
  },
  timeColumn: {
    width: 40,
    display: "flex",
    justifyContent: "center",
    alignItems: "baseline",
    paddingTop: 4,
  },
  smallTime: {
    fontSize: 12,
    color: "#888",
    width: 40,
    flexShrink: 0,
    marginTop: 4,
    textAlign: "center",
  },

  // Вложения в сообщении
  attachments: {
    marginTop: 6,
    display: "flex",
    flexDirection: "column",
    gap: 6,
  },
  inlineImage: {
    maxWidth: 320,
    maxHeight: 240,
    borderRadius: 8,
    objectFit: "cover",
    cursor: "zoom-in",
    display: "block",
  },
  inlineVideo: {
    maxWidth: "60%",
    maxHeight: 360,
    borderRadius: 8,
    display: "block",
    outline: "none",
    marginTop: 6,
  },
  fileLink: {
    display: "inline-flex",
    alignItems: "center",
    gap: 6,
    padding: "6px 10px",
    borderRadius: 8,
    background: "#f0f0f0",
    textDecoration: "none",
    color: "#333",
    fontSize: 14,
    maxWidth: 320,
  },
  fileIcon: { flexShrink: 0 },
  fileName: {
    overflow: "hidden",
    textOverflow: "ellipsis",
    whiteSpace: "nowrap",
    flex: 1,
  },
  fileSize: {
    fontSize: 12,
    color: "#888",
    flexShrink: 0,
  },

  // Лайтбокс
  lightboxOverlay: {
    position: "fixed",
    inset: 0,
    background: "rgba(0,0,0,0.85)",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    zIndex: 1000,
    cursor: "zoom-out",
  },
  lightboxImage: {
    maxWidth: "90vw",
    maxHeight: "90vh",
    objectFit: "contain",
    borderRadius: 4,
  },
  lightboxClose: {
    position: "absolute",
    top: 16,
    right: 16,
    background: "rgba(255,255,255,0.15)",
    border: "none",
    color: "#fff",
    fontSize: 20,
    width: 36,
    height: 36,
    borderRadius: "50%",
    cursor: "pointer",
  },

  // Превью файлов перед отправкой
  pendingFiles: {
    display: "flex",
    flexWrap: "wrap",
    gap: 8,
    padding: "8px 4px",
    borderTop: "1px solid #eee",
  },
  pendingItem: {
    position: "relative",
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    gap: 4,
    width: 72,
  },
  pendingPreview: {
    width: 64,
    height: 64,
    objectFit: "cover",
    borderRadius: 8,
    border: "1px solid #ddd",
  },
  pendingFileIcon: {
    width: 64,
    height: 64,
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    fontSize: 28,
    background: "#f0f0f0",
    borderRadius: 8,
    border: "1px solid #ddd",
  },
  pendingFileName: {
    fontSize: 11,
    color: "#555",
    textAlign: "center",
    wordBreak: "break-all",
    lineHeight: 1.2,
  },
  pendingRemove: {
    position: "absolute",
    top: -6,
    right: -6,
    width: 18,
    height: 18,
    borderRadius: "50%",
    background: "#ff4444",
    color: "#fff",
    border: "none",
    cursor: "pointer",
    fontSize: 10,
    padding: 0,
  },

  // Прогресс загрузки
  progressWrapper: {
    height: 4,
    background: "#e0e0e0",
    borderRadius: 2,
    margin: "4px 0",
    position: "relative",
    overflow: "hidden",
  },
  progressFill: {
    height: "100%",
    background: "#4a90e2",
    borderRadius: 2,
    transition: "width 0.1s",
  },
  progressText: {
    position: "absolute",
    right: 4,
    top: -14,
    fontSize: 11,
    color: "#888",
  },

  // Поле ввода
  inputRow: {
    display: "flex",
    gap: 8,
    alignItems: "center",
  },
  attachBtn: {
    padding: "8px 10px",
    fontSize: 18,
    background: "#f0f0f0",
    border: "1px solid #ddd",
    borderRadius: 8,
    cursor: "pointer",
    flexShrink: 0,
  },
  textInput: {
    flex: 1,
    padding: 10,
    fontSize: 16,
    border: "1px solid #ddd",
    borderRadius: 8,
    outline: "none",
  },
  sendBtn: {
    padding: "10px 18px",
    background: "#4a90e2",
    color: "#fff",
    border: "none",
    borderRadius: 8,
    cursor: "pointer",
    fontSize: 15,
    flexShrink: 0,
  },
  // Сетка изображений
  imageGrid: {
    display: "grid",
    gap: 4,
    marginTop: 6,
  },
  // Одно изображение — крупное
  singleImage: {
    maxWidth: "60%",
    maxHeight: 360,
    borderRadius: 8,
    objectFit: "cover" as const,
    cursor: "zoom-in",
    display: "block",
  },
  // Несколько изображений — квадратные thumbnail
  gridImage: {
    width: 80,
    height: 80,
    borderRadius: 6,
    objectFit: "cover" as const,
    cursor: "zoom-in",
    display: "block",
  },
  // Счётчик файлов
  fileCounter: {
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    fontSize: 12,
    color: "#888",
    padding: "0 4px",
    alignSelf: "center" as const,
  },
  bar: {
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    padding: "8px 12px",
    borderBottom: "1px solid #ddd",
    background: "#fff",
    flexShrink: 0,
    gap: 8,
  },
  userInfo: {
    display: "flex",
    alignItems: "center",
    gap: 10,
    overflow: "hidden",
  },
  actions: {
    display: "flex",
    alignItems: "center",
    gap: 4,
    flexShrink: 0,
  },
  searchInput: {
    padding: "6px 10px",
    fontSize: 14,
    border: "1px solid #ddd",
    borderRadius: 8,
    outline: "none",
    width: 200,
  },
  iconBtn: {
    background: "none",
    border: "none",
    cursor: "pointer",
    fontSize: 18,
    padding: "4px 6px",
    borderRadius: 6,
    lineHeight: 1,
  },
  // Модалка редактирования
  modalOverlay: {
    position: "fixed",
    inset: 0,
    background: "rgba(0,0,0,0.4)",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    zIndex: 100,
  },

  modalContent: {
    background: "#fff",
    borderRadius: 12,
    padding: 24,
    width: 400,
    display: "flex",
    flexDirection: "column",
    gap: 12,
  },

  modalTitle: {
    fontWeight: 600,
  },

  modalTextarea: {
    padding: 8,
    borderRadius: 8,
    border: "1px solid #ddd",
    fontSize: 14,
    resize: "vertical" as const,
  },

  modalActions: {
    display: "flex",
    gap: 8,
    justifyContent: "flex-end",
  },

  modalCancelBtn: {
    padding: "6px 16px",
    borderRadius: 8,
    border: "1px solid #ddd",
    cursor: "pointer",
    background: "#fff",
  },

  modalSaveBtn: {
    padding: "6px 16px",
    borderRadius: 8,
    border: "none",
    cursor: "pointer",
    background: "#4a90e2",
    color: "#fff",
  },
  bubbleWrapper: {
    position: "relative",
    transition: "background 0.2s",
  },

  bubbleHover: {
    background: "#c9daed",
  },

  bubbleHighlight: {
    background: "#c8f6f0",
  },

  // reply
  replyBlock: {
    borderLeft: "3px solid #888",
    background: "rgba(180,180,180,0.12)",
    borderRadius: "0 6px 6px 0",
    padding: "6px 8px",
    marginBottom: 4,
    fontSize: 13,
    cursor: "pointer",
  },

  replyAuthor: {
    fontWeight: 600,
    marginBottom: 2,
    opacity: 0.8,
  },

  replyText: {
    opacity: 0.7,
  },

  // forwarded
  forwardBlock: {
    borderLeft: "3px solid #A4C7F0",
    background: "rgba(164,199,240,0.15)",
    borderRadius: "0 6px 6px 0",
    padding: "6px 8px",
    marginBottom: 4,
    fontSize: 13,
  },

  forwardAuthor: {
    fontWeight: 600,
    marginBottom: 2,
    opacity: 0.8,
  },

  forwardText: {
    opacity: 0.7,
  },

  forwardAttachments: {
    marginTop: 4,
    display: "flex",
    flexWrap: "wrap",
    gap: 4,
  },

  forwardImage: {
    maxWidth: 120,
    maxHeight: 80,
    borderRadius: 4,
    cursor: "pointer",
    objectFit: "cover" as const,
  },

  forwardVideo: {
    maxWidth: 200,
    borderRadius: 4,
  },

  forwardFileLink: {
    display: "flex",
    alignItems: "center",
    gap: 4,
    fontSize: 12,
    opacity: 0.8,
  },

  // deleted
  deletedMessage: {
    opacity: 0.5,
    fontStyle: "italic",
  },

  editedLabel: {
    fontSize: 11,
    opacity: 0.5,
  },

  // actions
  actionsOverlay: {
    position: "absolute",
    top: 4,
    right: 8,
    zIndex: 10,
    display: "flex",
    alignItems: "center",
    gap: 4,
  },

  actionBtn: {
    background: "rgba(148,144,144,0.92)",
    border: "none",
    borderRadius: 8,
    cursor: "pointer",
    padding: "2px 8px",
    fontSize: 16,
    lineHeight: "22px",
  },

  menuWrapper: {
    position: "relative",
  },

  portalOverlay: {
    position: "fixed",
    inset: 0,
    zIndex: 9998,
  },

  dropdownMenu: {
    position: "fixed",
    background: "#fff",
    borderRadius: 8,
    boxShadow: "0 2px 12px rgba(0,0,0,0.15)",
    minWidth: 160,
    zIndex: 9999,
    overflow: "hidden",
  },

  dropdownItem: {
    padding: "8px 16px",
    cursor: "pointer",
    fontSize: 14,
    color: "#222",
  },

  dropdownItemDanger: {
    color: "#e53935",
  },

  unreadDivider: {
    textAlign: "center" as const,
    fontSize: 12,
    color: "#888",
    padding: "4px 0",
    margin: "8px 16px",
    borderTop: "1px solid #e0e0e0",
    position: "relative" as const,
  },
};
