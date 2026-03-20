// sozvon-client/src/components/chat/hooks/useVisibleMessages.ts

import { useRef, useEffect, useCallback } from "react";

export function useVisibleMessages(onMarkRead: (id: string) => void) {
  const lastVisibleRef = useRef<{ id: string; time: number } | null>(null);
  const observerRef = useRef<IntersectionObserver | null>(null);

  const observe = useCallback(
    (el: HTMLDivElement | null, id: string, createdAt: string) => {
      if (!el || !observerRef.current) return;
      el.dataset.messageId = id;
      el.dataset.messageTime = String(new Date(createdAt).getTime());
      observerRef.current.observe(el);
    },
    [],
  );

  useEffect(() => {
    observerRef.current = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (!entry.isIntersecting) return;
          const id = (entry.target as HTMLElement).dataset.messageId;
          const time = Number(
            (entry.target as HTMLElement).dataset.messageTime,
          );
          if (!id || !time) return;

          // Обновляем только если сообщение новее текущего максимума
          if (!lastVisibleRef.current || time > lastVisibleRef.current.time) {
            lastVisibleRef.current = { id, time };
          }
        });
      },
      { threshold: 0.8 },
    );

    return () => {
      flush();
      observerRef.current?.disconnect();
    };
  }, []);

  const flush = useCallback(() => {
    if (lastVisibleRef.current) {
      onMarkRead(lastVisibleRef.current.id);
      lastVisibleRef.current = null;
    }
  }, [onMarkRead]);

  const reset = useCallback(() => {
    lastVisibleRef.current = null;
    observerRef.current?.disconnect();
  }, []);

  return { observe, flush, reset };
}
