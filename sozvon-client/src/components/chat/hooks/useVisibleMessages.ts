// sozvon-client/src/components/chat/hooks/useVisibleMessages.ts

import { useRef, useEffect, useCallback } from "react";

type MessageMeta = {
  id: string;
  time: number;
};

export function useVisibleMessages(onMarkRead: (id: string) => void) {
  const observerRef = useRef<IntersectionObserver | null>(null);
  const elementsRef = useRef<Set<HTMLElement>>(new Set());
  const visibleIdsRef = useRef<Set<string>>(new Set());
  const onMarkReadRef = useRef(onMarkRead);
  const maxReadTimeRef = useRef<Map<string, MessageMeta>>(new Map());
  const activeChatIdRef = useRef<string>("");

  const setActiveChatId = useCallback((id: string) => {
    activeChatIdRef.current = id;
  }, []);

  useEffect(() => {
    onMarkReadRef.current = onMarkRead;
  }, [onMarkRead]);

  const handleIntersect = useCallback(
    (entries: IntersectionObserverEntry[]) => {
      for (const entry of entries) {
        const el = entry.target as HTMLElement;
        const id = el.dataset.messageId;

        if (!id) continue;

        if (entry.isIntersecting) {
          visibleIdsRef.current.add(id);
        } else {
          visibleIdsRef.current.delete(id);
        }
      }

      // 👉 realtime markRead
      const allElements = Array.from(elementsRef.current);

      let best: MessageMeta | null = null;

      for (const id of visibleIdsRef.current) {
        const el = allElements.find((e) => e.dataset.messageId === id);
        if (!el) continue;

        const time = Number(el.dataset.messageTime);
        if (Number.isNaN(time)) continue;

        if (!best || time > best.time) {
          best = { id, time };
        }
      }

      if (!best) return;

      const chatId = activeChatIdRef.current;
      if (!chatId) return;

      const prev = maxReadTimeRef.current.get(chatId);

      if (!prev || best.time > prev.time) {
        maxReadTimeRef.current.set(chatId, best);
      }
    },
    [],
  );

  const createObserver = useCallback(() => {
    observerRef.current?.disconnect();

    observerRef.current = new IntersectionObserver(handleIntersect, {
      threshold: 0.8,
    });

    elementsRef.current.forEach((el) => {
      observerRef.current?.observe(el);
    });
  }, [handleIntersect]);

  const flush = useCallback((chatId: string) => {
    console.log(
      `[flush] elementsRef.size=${elementsRef.current.size}, visibleIds.size=${visibleIdsRef.current.size}`,
    );

    const best = maxReadTimeRef.current.get(chatId);

    if (!best) {
      console.log(
        `[useVisibleMessages] flush chatId=${chatId}: nothing seen, skipping`,
      );
      return;
    }

    console.log(
      `[useVisibleMessages] flush chatId=${chatId}: sending read to id="${best.id}" time=${new Date(best.time).toISOString()}`,
    );
    onMarkReadRef.current(best.id);
  }, []);

  useEffect(() => {
    createObserver();

    return () => {
      observerRef.current?.disconnect();
      elementsRef.current.clear();
      visibleIdsRef.current.clear();
    };
  }, [createObserver]);

  const observe = useCallback(
    (el: HTMLDivElement | null, id: string, createdAt: string) => {
      if (!el) return;

      const time = new Date(createdAt).getTime();
      if (Number.isNaN(time)) return;

      el.dataset.messageId = id;
      el.dataset.messageTime = String(time);

      if (!elementsRef.current.has(el)) {
        elementsRef.current.add(el);
        observerRef.current?.observe(el);
      }
    },
    [],
  );

  const unobserve = useCallback((el: HTMLDivElement | null) => {
    if (!el) return;

    elementsRef.current.delete(el);
    observerRef.current?.unobserve(el);
  }, []);

  const reset = useCallback(() => {
    visibleIdsRef.current.clear();
    elementsRef.current.clear();
    observerRef.current?.disconnect();
    createObserver();
  }, [createObserver]);

  const initChat = useCallback((chatId: string) => {
    maxReadTimeRef.current.delete(chatId);
    console.log(
      `[useVisibleMessages] initChat: reset maxReadTime for ${chatId}`,
    );
  }, []);

  return { observe, unobserve, flush, reset, initChat, setActiveChatId };
}
