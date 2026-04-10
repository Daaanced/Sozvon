// sozvon-client/src/components/chat/hooks/useVisibleMessages.ts

import { useRef, useEffect, useCallback } from "react";

export function useVisibleMessages(onMarkRead: (id: string) => void) {

  const observerRef = useRef<IntersectionObserver | null>(null);
  const elementsRef = useRef<Set<HTMLElement>>(new Set());
  const visibleIdsRef = useRef<Set<string>>(new Set());
  const onMarkReadRef = useRef(onMarkRead);
  const maxReadRef = useRef<{ id: string; time: number } | null>(null);

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

  // Обновляем максимум среди текущих видимых
  const allElements = Array.from(elementsRef.current);
  for (const id of visibleIdsRef.current) {
    const el = allElements.find((e) => e.dataset.messageId === id);
    if (!el) continue;
    const time = Number(el.dataset.messageTime);
    if (Number.isNaN(time)) continue;

    // Накапливаем максимум за всё время — не сбрасываем при скролле вверх
    if (!maxReadRef.current || time > maxReadRef.current.time) {
      maxReadRef.current = { id, time };
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

	const flush = useCallback(() => {
		// Берём накопленный максимум, а не текущие видимые
		const best = maxReadRef.current;
		if (!best) return;
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
	maxReadRef.current = null; // ← сброс при смене чата
	observerRef.current?.disconnect();
	createObserver();
	}, [createObserver]);

  return { observe, unobserve, flush, reset };
}
