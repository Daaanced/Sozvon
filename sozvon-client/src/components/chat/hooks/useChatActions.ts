// sozvon-client/src/components/chat/hooks/useChatActions.ts
import { useState, useCallback } from "react";
import {
  sendMessage,
  uploadFiles,
  editMessage,
  deleteMessage,
} from "../../../api/chats";
import { useChatContext } from "../../../context/ChatContext";
import type { Message, PendingFile, Attachment } from "../chat.types";

interface SendParams {
  text: string;
  pendingFiles: PendingFile[];
  replyTo: Message | null;
  onClear: () => void;
}

export function useChatActions(
  chatId: string,
  setMessages: React.Dispatch<React.SetStateAction<Message[]>>,
) {
  const { notifyOwnMessage } = useChatContext();

  const [replyTo, setReplyTo] = useState<Message | null>(null);
  const [editingMessage, setEditingMessage] = useState<Message | null>(null);
  const [editText, setEditText] = useState("");

  const handleEdit = useCallback((msg: Message) => {
    setEditingMessage(msg);
    setEditText(msg.text);
  }, []);

  const cancelEdit = useCallback(() => {
    setEditingMessage(null);
    setEditText("");
  }, []);

  const submitEdit = useCallback(async () => {
    if (!editingMessage) return;
    try {
      await editMessage(chatId, editingMessage.id, editText);
      setMessages((prev) =>
        prev.map((m) =>
          m.id === editingMessage.id
            ? { ...m, text: editText, editedAt: new Date().toISOString() }
            : m,
        ),
      );
      cancelEdit();
    } catch (e) {
      console.error("Edit failed:", e);
    }
  }, [chatId, editingMessage, editText, cancelEdit]);

  const handleDelete = useCallback(
    async (msg: Message) => {
      if (!confirm("Удалить сообщение?")) return;
      try {
        await deleteMessage(chatId, msg.id);
        setMessages((prev) => prev.filter((m) => m.id !== msg.id));
      } catch (e) {
        console.error("Delete failed:", e);
      }
    },
    [chatId],
  );

  const send = useCallback(
    async ({ text, pendingFiles, replyTo, onClear }: SendParams) => {
      const trimmed = text.trim();
      const hasFiles = pendingFiles.length > 0;
      if (!trimmed && !hasFiles) return;

      onClear();

      try {
        let msg: Message;

        if (hasFiles) {
          msg = await uploadFiles(chatId, trimmed, pendingFiles, replyTo?.id);

          if (msg.attachments) {
            const map = new Map(
              pendingFiles.map((pf) => [pf.file.name + pf.file.size, pf]),
            );

            msg.attachments = msg.attachments.map((att: Attachment) => {
              const key = att.fileName + att.size;
              const pf = map.get(key);

              return {
                ...att,
                width: pf?.width,
                height: pf?.height,
              };
            });
          }
        } else {
          msg = await sendMessage(chatId, trimmed, replyTo?.id);
        }

        setMessages((prev) => [...prev, msg]);
        notifyOwnMessage(chatId, msg.createdAt, msg.id);
        setReplyTo(null);
      } catch (e) {
        console.error("Send failed:", e);
        throw e;
      }
    },
    [chatId, notifyOwnMessage],
  );

  return {
    replyTo,
    setReplyTo,
    editingMessage,
    editText,
    setEditText,
    handleEdit,
    cancelEdit,
    submitEdit,
    handleDelete,
    send,
  };
}
