// sozvon-client/src/components/chat/hooks/useChatMessages.ts

import { useState, useEffect, useCallback, useRef } from "react";
import { onWSMessage } from "../../../services/ws";
import {
  getMessages,
  getMessagesContext,
  getUnreadMessages,
  getMessagesAfter,
  getMessagesBefore,
} from "../../../api/chats";
import { useChatContext } from "../../../context/ChatContext";
import type { Message } from "../chat.types";

const PAGE_SIZE = 50;

export type ScrollIntent =
  | { type: "bottom" }
  | { type: "unread"; id: string }
  | { type: "message"; id: string }
  | null;

export function useChatMessages(
  chatId: string,
  onInitChat: (chatId: string, time: number) => void,
) {
  const { markRead, loadUser } = useChatContext();
  const [messages, setMessages] = useState<Message[]>([]);
  const [scrollIntent, setScrollIntent] = useState<ScrollIntent>(null);
  const [highlightId, setHighlightId] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const messagesRef = useRef<Message[]>([]);
  const loadingMoreRef = useRef(false);
  const hasMoreRef = useRef(true);
  const [hasMoreBottom, setHasMoreBottom] = useState(false);
  const [loadingMoreBottom, setLoadingMoreBottom] = useState(false);
  const hasMoreBottomRef = useRef(false);
  const loadingMoreBottomRef = useRef(false);
  const pendingBottomRef = useRef<Message[]>([]);
  const jumpingToBottomRef = useRef(false);
  const [initialized, setInitialized] = useState(false);
  const scrollingToUnreadRef = useRef(false);
  const initializedRef = useRef(false);

  useEffect(() => {
  initializedRef.current = initialized;
}, [initialized]);
  useEffect(() => {
    hasMoreBottomRef.current = hasMoreBottom;
  }, [hasMoreBottom]);
  useEffect(() => {
    loadingMoreBottomRef.current = loadingMoreBottom;
  }, [loadingMoreBottom]);
  // Синхронизируем рефы с актуальным стейтом
  useEffect(() => {
    messagesRef.current = messages;
  }, [messages]);

  useEffect(() => {
    loadingMoreRef.current = loadingMore;
  }, [loadingMore]);

  useEffect(() => {
    hasMoreRef.current = hasMore;
  }, [hasMore]);

  // Загрузка при смене чата
  useEffect(() => {
    setMessages([]);
    setScrollIntent(null);
    setHighlightId(null);
    setHasMore(true);
    setLoadingMore(false);
	setInitialized(false);
	scrollingToUnreadRef.current = false;
    loadingMoreRef.current = false;
    hasMoreRef.current = true;
    loadingMoreBottomRef.current = false;
    hasMoreBottomRef.current = false;
	pendingBottomRef.current = [];
	initializedRef.current = false;
    let cancelled = false;

    (async () => {
      try {
        const unread = await getUnreadMessages(chatId);
        console.log(
          `[useChatMessages][${chatId}] unread response: hasMoreTop=${unread.hasMoreTop}, hasMoreBottom=${unread.hasMoreBottom}, count=${unread.messages?.length}`,
        );
        if (cancelled) return;

        if (Array.isArray(unread.messages) && unread.messages.length > 0) {
          setMessages(unread.messages);
          setHasMore(unread.hasMoreTop === true);
          setHasMoreBottom(unread.hasMoreBottom === true);
          setScrollIntent({ type: "unread", id: unread.firstUnreadId });
		  scrollingToUnreadRef.current = true;
		  
          if (unread.messages.length > 0) {
            const lastMsg = unread.messages[unread.messages.length - 1];
            onInitChat(chatId, new Date(lastMsg.createdAt).getTime());
          }
		  setInitialized(true);
        } else {
          const data = await getMessages(chatId, PAGE_SIZE, 0);
          if (cancelled) return;
          const msgs: Message[] = Array.isArray(data) ? data : [];
          setMessages(msgs);
          setHasMore(msgs.length >= PAGE_SIZE);
          setScrollIntent({ type: "bottom" });

          if (msgs.length > 0) {
            const lastMsg = msgs[msgs.length - 1];
            onInitChat(chatId, new Date(lastMsg.createdAt).getTime());
          }
          msgs.forEach((m) => {
            if (m.forwardedFrom?.senderId) loadUser(m.forwardedFrom.senderId);
          });
		  setInitialized(true);
        }
      } catch (e) {
        console.error(`[useChatMessages][${chatId}] initial load failed:`, e);
        if (!cancelled) {
          setHasMoreBottom(false);
          setScrollIntent({ type: "bottom" });
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [chatId]);

  // Входящие сообщения по WebSocket
  useEffect(() => {
  return onWSMessage((msg) => {
    if (msg.event === "message:new" && msg.data.chatId === chatId) {
      if (msg.data.forwardedFrom?.senderId) {
        loadUser(msg.data.forwardedFrom.senderId);
      }
      markRead(chatId, msg.data.id);

      if (hasMoreBottomRef.current) {
        // Есть непрогруженные сообщения внизу — буферизуем
        pendingBottomRef.current = [...pendingBottomRef.current, msg.data];
      } else {
        // Находимся на актуальном конце — добавляем сразу
        setMessages((prev) => [...prev, msg.data]);
      }
    }
  });
}, [chatId]);

useEffect(() => {
  if (hasMoreBottom) return;

  const pending = pendingBottomRef.current;
  if (pending.length === 0) return;
  pendingBottomRef.current = [];

  setMessages((prev) => {
    const existingIds = new Set(prev.map((m) => m.id));
    const fresh = pending.filter((m) => !existingIds.has(m.id));
    if (fresh.length === 0) return prev;
    return [...prev, ...fresh].sort(
      (a, b) =>
        new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
    );
  });
}, [hasMoreBottom]);

const jumpToBottom = useCallback(async () => {
  if (jumpingToBottomRef.current) return;

  if (!hasMoreBottomRef.current) {
    setScrollIntent({ type: "bottom" });
    return;
  }

  jumpingToBottomRef.current = true;

  // Сбрасываем буфер входящих
  pendingBottomRef.current = [];

  try {
    const data = await getMessages(chatId, PAGE_SIZE, 0);
    const msgs: Message[] = Array.isArray(data) ? data : [];

    setMessages(msgs);
    setHasMore(msgs.length >= PAGE_SIZE);
    setHasMoreBottom(false);
    setScrollIntent({ type: "bottom" });
  } catch (e) {
    console.error(`[useChatMessages][${chatId}] jumpToBottom failed:`, e);
  } finally {
    jumpingToBottomRef.current = false;
  }
}, [chatId]);

  const loadMoreBottom = useCallback(async () => {
	console.log(`[loadMoreBottom] called: loading=${loadingMoreBottomRef.current}, hasMore=${hasMoreBottomRef.current}, scrollingToUnread=${scrollingToUnreadRef.current}`);
    if (loadingMoreBottomRef.current || !hasMoreBottomRef.current) return;
	if (!initializedRef.current) return;
	if (scrollingToUnreadRef.current) return;

    const msgs = messagesRef.current;
    if (msgs.length === 0) return;

    const lastMsg = msgs[msgs.length - 1];
    setLoadingMoreBottom(true);

    try {
      // запрашиваем сообщения после последнего известного
      const data = await getMessagesAfter(chatId, lastMsg.id, PAGE_SIZE);
      const newer: Message[] = Array.isArray(data) ? data : [];

      if (newer.length === 0) {
        setHasMoreBottom(false);
        return;
      }

      newer.forEach((m) => {
        if (m.forwardedFrom?.senderId) loadUser(m.forwardedFrom.senderId);
      });

      setMessages((current) => {
        const existingIds = new Set(current.map((m) => m.id));
        const fresh = newer.filter((m) => !existingIds.has(m.id));
        return [...current, ...fresh].sort(
          (a, b) =>
            new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
        );
      });

      if (newer.length < PAGE_SIZE) setHasMoreBottom(false);
    } catch (e) {
      console.error(`[useChatMessages][${chatId}] loadMoreBottom failed:`, e);
    } finally {
      setLoadingMoreBottom(false);
    }
  }, [chatId]);

  // Подгрузка старых сообщений (скролл вверх)
  const loadMore = useCallback(async () => {
    if (loadingMoreRef.current || !hasMoreRef.current) return;
    if (!initializedRef.current) return;

    const msgs = messagesRef.current;
    if (msgs.length === 0) return;

    const firstMsg = msgs[0]; // ← берём первое сообщение, а не offset
    setLoadingMore(true);

    try {
      const data = await getMessagesBefore(chatId, firstMsg.id, PAGE_SIZE);
      const older: Message[] = Array.isArray(data) ? data : [];

      if (older.length === 0) {
        setHasMore(false);
        return;
      }

      older.forEach((m) => {
        if (m.forwardedFrom?.senderId) loadUser(m.forwardedFrom.senderId);
      });

      setMessages((current) => {
        const existingIds = new Set(current.map((m) => m.id));
        const fresh = older.filter((m) => !existingIds.has(m.id));
        return [...fresh, ...current].sort(
          (a, b) =>
            new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
        );
      });

      if (older.length < PAGE_SIZE) setHasMore(false);
    } catch (e) {
      console.error(`[useChatMessages][${chatId}] loadMore failed:`, e);
    } finally {
      setLoadingMore(false);
    }
  }, [chatId]);

  // Скролл к сообщению (с подгрузкой контекста если нужно)
  const scrollToMessage = useCallback(
    async (
      id: string,
      messageRefs: React.MutableRefObject<
        Record<string, HTMLDivElement | null>
      >,
    ) => {
      if (messageRefs.current[id]) {
        setScrollIntent({ type: "message", id });
        setHighlightId(id);
        setTimeout(() => setHighlightId(null), 1500);
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

        setScrollIntent({ type: "message", id });
        setHighlightId(id);
        setTimeout(() => setHighlightId(null), 1500);
      } catch (e) {
        console.error(
          `[useChatMessages][${chatId}] scrollToMessage failed:`,
          e,
        );
      }
    },
    [chatId],
  );

  const consumeScrollIntent = useCallback(() => {
    setScrollIntent(null);
  }, []);

  const clearScrollingToUnread = useCallback(() => {
  console.log(`[useChatMessages][${chatId}] clearScrollingToUnread called`);
  scrollingToUnreadRef.current = false;
}, [chatId]);

  return {
    messages,
    setMessages,
    scrollIntent,
    consumeScrollIntent,
    highlightId,
    hasMore,
    loadingMore,
    loadMore,
    hasMoreBottom,
    loadingMoreBottom,
    loadMoreBottom,
    scrollToMessage,
	jumpToBottom,
	initialized,
	clearScrollingToUnread,
  };
}
