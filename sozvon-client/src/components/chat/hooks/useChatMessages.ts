// sozvon-client/src/components/chat/hooks/useChatMessages.ts

import { useState, useEffect, useCallback } from "react";
import { onWSMessage } from "../../../services/ws";
import {
  getMessages,
  getMessagesContext,
  getUnreadMessages,
} from "../../../api/chats";
import { useChatContext } from "../../../context/ChatContext";
import type { Message } from "../chat.types";
 
const PAGE_SIZE = 50;
 
export function useChatMessages(chatId: string) {
  const { markRead } = useChatContext();
  const [messages, setMessages] = useState<Message[]>([]);
  const [firstUnreadId, setFirstUnreadId] = useState<string | null>(null);
  const [highlightId, setHighlightId] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
 
  // Загрузка при смене чата
  useEffect(() => {
    setMessages([]);
    setFirstUnreadId(null);
    setHasMore(true);
 
    (async () => {
      const unread = await getUnreadMessages(chatId);
 
      if (unread.firstUnreadId && Array.isArray(unread.messages)) {
        setMessages(unread.messages);
        setFirstUnreadId(unread.firstUnreadId);
        setHasMore(unread.messages.length >= PAGE_SIZE);
      } else {
        const data = await getMessages(chatId, PAGE_SIZE, 0);
        const msgs: Message[] = Array.isArray(data) ? data : [];
        setMessages(msgs);
        setHasMore(msgs.length >= PAGE_SIZE);
        if (msgs.length > 0) {
          markRead(chatId, msgs[msgs.length - 1].id);
        }
      }
    })();
  }, [chatId]);
 
  // Входящие сообщения по WebSocket
  useEffect(() => {
    return onWSMessage((msg) => {
      if (msg.event === "message:new" && msg.data.chatId === chatId) {
        setMessages((prev) => [...prev, msg.data]);
        markRead(chatId, msg.data.id);
      }
    });
  }, [chatId]);
 
  // Подгрузка старых сообщений (скролл вверх)
  const loadMore = useCallback(async () => {
    if (loadingMore || !hasMore) return;
    setLoadingMore(true);
 
    setMessages((prev) => {
      const offset = prev.length;
      getMessages(chatId, PAGE_SIZE, offset)
        .then((data) => {
          const older: Message[] = Array.isArray(data) ? data : [];
          if (older.length === 0) {
            setHasMore(false);
            return;
          }
          setMessages((current) => {
            const existingIds = new Set(current.map((m) => m.id));
            const fresh = older.filter((m) => !existingIds.has(m.id));
            return [...fresh, ...current].sort(
              (a, b) =>
                new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
            );
          });
          if (older.length < PAGE_SIZE) setHasMore(false);
        })
        .catch((e) => console.error("loadMore failed:", e))
        .finally(() => setLoadingMore(false));
      return prev;
    });
  }, [chatId, loadingMore, hasMore]);
 
  // Скролл к сообщению (с подгрузкой контекста если нужно)
  const scrollToMessage = useCallback(
    async (
      id: string,
      messageRefs: React.MutableRefObject<Record<string, HTMLDivElement | null>>,
    ) => {
      const scrollAndHighlight = () => {
        messageRefs.current[id]?.scrollIntoView({ behavior: "smooth", block: "center" });
        setHighlightId(id);
        setTimeout(() => setHighlightId(null), 1500);
      };
 
      if (messageRefs.current[id]) {
        scrollAndHighlight();
        return;
      }
 
      try {
        const data = await getMessagesContext(chatId, id);
        if (!Array.isArray(data)) return;
 
        setMessages((prev) => {
          const existingIds = new Set(prev.map((m) => m.id));
          return [
            ...prev,
            ...data.filter((m: Message) => !existingIds.has(m.id)),
          ].sort(
            (a, b) =>
              new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
          );
        });
 
        setHighlightId(id);
        setTimeout(() => scrollAndHighlight(), 100);
      } catch (e) {
        console.error("Failed to load message context:", e);
      }
    },
    [chatId],
  );
 
  return {
    messages,
    setMessages,
    firstUnreadId,
    setFirstUnreadId,
    highlightId,
    hasMore,
    loadingMore,
    loadMore,
    scrollToMessage,
  };
}