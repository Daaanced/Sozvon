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
  const maxReadTimeRef = useRef<Map<string, number>>(new Map());

  useEffect(() => {
    onMarkReadRef.current = onMarkRead;
  }, [onMarkRead]);

  const handleIntersect = useCallback((entries: IntersectionObserverEntry[]) => {
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
  }, []);

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
  const visibleIds = Array.from(visibleIdsRef.current);
  const allElements = Array.from(elementsRef.current);

  let bestVisible: MessageMeta | null = null;

  if (visibleIds.length === 0) {
    const last = allElements[allElements.length - 1];
    if (last?.dataset.messageId) {
      bestVisible = {
        id: last.dataset.messageId,
        time: Number(last.dataset.messageTime),
      };
    }
  } else {
    for (const id of visibleIds) {
      const el = allElements.find((e) => e.dataset.messageId === id);
      if (!el) continue;
      const time = Number(el.dataset.messageTime);
      if (Number.isNaN(time)) continue;
      if (!bestVisible || time > bestVisible.time) {
        bestVisible = { id, time };
      }
    }
  }

  if (!bestVisible) return;

  const prevMax = maxReadTimeRef.current.get(chatId) ?? 0;

  console.log(`[useVisibleMessages] flush chatId=${chatId}: bestVisible=${new Date(bestVisible.time).toISOString()}, prevMax=${prevMax ? new Date(prevMax).toISOString() : "none"}`);

  // Отправляем только если новая позиция НОВЕЕ сохранённой
  if (bestVisible.time > prevMax) {
    maxReadTimeRef.current.set(chatId, bestVisible.time);
    console.log(`[useVisibleMessages] flush: advancing read to id="${bestVisible.id}"`);
    onMarkReadRef.current(bestVisible.id);
  } else {
    console.log(`[useVisibleMessages] flush: skipping — user scrolled up, keeping max position`);
  }
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

  const initChat = useCallback((chatId: string, time: number) => {
  const current = maxReadTimeRef.current.get(chatId) ?? 0;
  if (time > current) {
    console.log(`[useVisibleMessages] initChat: setting maxReadTime for ${chatId} to ${new Date(time).toISOString()}`);
    maxReadTimeRef.current.set(chatId, time);
  }
}, []);

return { observe, unobserve, flush, reset, initChat };

}