// sozvon-client/src/components/chat/hooks/useChatMessages.ts

import { useState, useEffect, useCallback, useRef } from "react";
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
  const [initialLoading, setInitialLoading] = useState(true);
  const messagesRef = useRef<Message[]>([]);
  const initialLoadingRef = useRef(true);

  // Синхронизируем ref с актуальным стейтом
  useEffect(() => {
  messagesRef.current = messages;
}, [messages]);

  // Загрузка при смене чата
  useEffect(() => {
	initialLoadingRef.current = true;
    console.log(`[useChatMessages] chatId changed → "${chatId}". Resetting state.`);
    setMessages([]);
    setFirstUnreadId(null);
    setHasMore(true);
	setInitialLoading(true);

    let cancelled = false;

    (async () => {
      console.log(`[useChatMessages][${chatId}] Fetching unread messages...`);
      const unread = await getUnreadMessages(chatId);

      if (cancelled) {
        console.warn(`[useChatMessages][${chatId}] Request cancelled (chatId changed during getUnreadMessages). Ignoring result.`);
        return;
      }

      console.log(`[useChatMessages][${chatId}] getUnreadMessages result:`, {
        firstUnreadId: unread?.firstUnreadId,
        messagesCount: Array.isArray(unread?.messages) ? unread.messages.length : "not array",
        raw: unread,
      });

      if (unread.firstUnreadId && Array.isArray(unread.messages)) {
        console.log(`[useChatMessages][${chatId}] Has unread. Setting ${unread.messages.length} messages. firstUnreadId="${unread.firstUnreadId}"`);
        setMessages(unread.messages);
        setFirstUnreadId(unread.firstUnreadId);
        setHasMore(unread.messages.length >= PAGE_SIZE);
      } else {
        console.log(`[useChatMessages][${chatId}] No unread. Fetching latest messages...`);
        const data = await getMessages(chatId, PAGE_SIZE, 0);

        if (cancelled) {
          console.warn(`[useChatMessages][${chatId}] Request cancelled (chatId changed during getMessages). Ignoring result.`);
          return;
        }

        const msgs: Message[] = Array.isArray(data) ? data : [];
        console.log(`[useChatMessages][${chatId}] getMessages result: ${msgs.length} messages.`);

        if (msgs.length > 0) {
          console.log(`[useChatMessages][${chatId}] First msg:`, msgs[0]?.id, msgs[0]?.createdAt);
          console.log(`[useChatMessages][${chatId}] Last msg:`, msgs[msgs.length - 1]?.id, msgs[msgs.length - 1]?.createdAt);
        }

        setMessages(msgs);
        setHasMore(msgs.length >= PAGE_SIZE);
      }
	  if (!cancelled) {
      setInitialLoading(false); // ← только после успешной загрузки
	  initialLoadingRef.current = false;
    }
	
    })();

    return () => {
      console.log(`[useChatMessages] Cleanup for chatId="${chatId}". Setting cancelled=true.`);
      cancelled = true;
    };
  }, [chatId]);

  // Входящие сообщения по WebSocket
  useEffect(() => {
    return onWSMessage((msg) => {
      if (msg.event === "message:new" && msg.data.chatId === chatId) {
        console.log(`[useChatMessages][${chatId}] WS message:new received:`, msg.data.id, msg.data.createdAt);
        setMessages((prev) => [...prev, msg.data]);
        markRead(chatId, msg.data.id);
      }
    });
  }, [chatId]);

  // Подгрузка старых сообщений (скролл вверх)
  const loadMore = useCallback(async () => {
	console.log(`[loadMore] called. loadingMore=${loadingMore}, hasMore=${hasMore}, initialLoadingRef=${initialLoadingRef.current}`);
    if (loadingMore || !hasMore || initialLoadingRef.current) return;

    const offset = messagesRef.current.length;
    console.log(`[useChatMessages][${chatId}] loadMore called. offset=${offset}, hasMore=${hasMore}`);

    setLoadingMore(true);

    try {
      const data = await getMessages(chatId, PAGE_SIZE, offset);
      const older: Message[] = Array.isArray(data) ? data : [];

      console.log(`[useChatMessages][${chatId}] loadMore got ${older.length} older messages.`);

      if (older.length === 0) {
        console.log(`[useChatMessages][${chatId}] loadMore: no more messages. setHasMore(false).`);
        setHasMore(false);
        return;
      }

      if (older.length > 0) {
        console.log(`[useChatMessages][${chatId}] loadMore older range: ${older[0]?.createdAt} → ${older[older.length - 1]?.createdAt}`);
      }

      setMessages((current) => {
        const existingIds = new Set(current.map((m) => m.id));
        const fresh = older.filter((m) => !existingIds.has(m.id));

        console.log(`[useChatMessages][${chatId}] loadMore merging: existing=${current.length}, fresh=${fresh.length}, duplicates=${older.length - fresh.length}`);

        const merged = [...fresh, ...current].sort(
          (a, b) =>
            new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
        );

        console.log(`[useChatMessages][${chatId}] loadMore after merge: total=${merged.length}`);
        console.log(`[useChatMessages][${chatId}] loadMore merged range: ${merged[0]?.createdAt} → ${merged[merged.length - 1]?.createdAt}`);

        return merged;
      });

      if (older.length < PAGE_SIZE) {
        console.log(`[useChatMessages][${chatId}] loadMore: received < PAGE_SIZE, setHasMore(false).`);
        setHasMore(false);
      }
    } catch (e) {
      console.error(`[useChatMessages][${chatId}] loadMore failed:`, e);
    } finally {
      setLoadingMore(false);
    }
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
        console.log(`[useChatMessages][${chatId}] scrollToMessage: id="${id}" already in DOM. Scrolling.`);
        scrollAndHighlight();
        return;
      }

      console.log(`[useChatMessages][${chatId}] scrollToMessage: id="${id}" NOT in DOM. Loading context...`);

      try {
        const data = await getMessagesContext(chatId, id);
        if (!Array.isArray(data)) return;

        console.log(`[useChatMessages][${chatId}] scrollToMessage context: got ${data.length} messages.`);

        setMessages((prev) => {
          const existingIds = new Set(prev.map((m) => m.id));
          const merged = [
            ...prev,
            ...data.filter((m: Message) => !existingIds.has(m.id)),
          ].sort(
            (a, b) =>
              new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
          );
          console.log(`[useChatMessages][${chatId}] scrollToMessage after merge: total=${merged.length}`);
          return merged;
        });

        setHighlightId(id);
        setTimeout(() => scrollAndHighlight(), 100);
      } catch (e) {
        console.error(`[useChatMessages][${chatId}] scrollToMessage failed:`, e);
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
	initialLoading,
  };
}