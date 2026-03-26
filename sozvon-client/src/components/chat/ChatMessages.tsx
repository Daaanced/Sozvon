// sozvon-client/src/components/chat/ChatMessages.tsx

import { useRef, useEffect, useLayoutEffect } from "react";
import MessageBubble from "./MessageBubble";
import { styles } from "./chat.styles";
import type { Message } from "./chat.types";
import type { User } from "../../api/users";
import type { ScrollIntent } from "./hooks/useChatMessages";

type Props = {
  chatId: string;
  messages: Message[];
  getUser: (senderId: number) => User;
  onImageClick: (url: string) => void;
  onReply: (message: Message) => void;
  onForward: (message: Message) => void;
  onEdit: (message: Message) => void;
  onDelete: (message: Message) => void;
  observeMessage: (
    el: HTMLDivElement | null,
    id: string,
    createdAt: string,
  ) => void;
  unobserveMessage: (el: HTMLDivElement | null) => void;
  myId: number;
  highlightId: string | null;
  onScrollToMessage: (
    id: string,
    messageRefs: React.MutableRefObject<Record<string, HTMLDivElement | null>>,
  ) => void;
  scrollIntent: ScrollIntent;
  onIntentConsumed: () => void;
  hasMore: boolean;
  loadingMore: boolean;
  onLoadMore: () => void;
  hasMoreBottom: boolean;
  loadingMoreBottom: boolean;
  onLoadMoreBottom: () => void;
};

function isSameDay(a: string, b: string) {
  const d1 = new Date(a),
    d2 = new Date(b);
  return (
    d1.getFullYear() === d2.getFullYear() &&
    d1.getMonth() === d2.getMonth() &&
    d1.getDate() === d2.getDate()
  );
}

export default function ChatMessages({
  chatId,
  messages,
  getUser,
  onImageClick,
  onReply,
  onForward,
  onEdit,
  onDelete,
  myId,
  highlightId,
  onScrollToMessage,
  scrollIntent,
  onIntentConsumed,
  observeMessage,
  unobserveMessage,
  hasMore,
  loadingMore,
  onLoadMore,
  hasMoreBottom,
  loadingMoreBottom,
  onLoadMoreBottom,
}: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const bottomSentinelRef = useRef<HTMLDivElement>(null);
  const messageRefs = useRef<Record<string, HTMLDivElement | null>>({});
  const prevScrollHeight = useRef<number | null>(null);
  const firstUnreadIdRef = useRef<string | null>(null);

  if (scrollIntent?.type === "unread" && !firstUnreadIdRef.current) {
    firstUnreadIdRef.current = scrollIntent.id;
  }

  // 1. Сброс при смене чата
  useEffect(() => {
    messageRefs.current = {};
    firstUnreadIdRef.current = null;
    if (ref.current) ref.current.scrollTop = 0;
  }, [chatId]);

  // 2. Применение scroll intent — useLayoutEffect чтобы DOM уже содержал новые сообщения
  useLayoutEffect(() => {
    if (!scrollIntent || messages.length === 0) return;

    const el = ref.current;
    if (!el) return;

    if (scrollIntent.type === "bottom") {
      el.scrollTop = el.scrollHeight;
      onIntentConsumed();
      return;
    }

    const target = messageRefs.current[scrollIntent.id];
    if (!target) return; // сообщение ещё не в DOM — ждём следующего рендера

    if (scrollIntent.type === "unread") {
      target.scrollIntoView({ behavior: "instant", block: "center" });
      setTimeout(onIntentConsumed, 100);
      return;
    } else if (scrollIntent.type === "message") {
      target.scrollIntoView({ behavior: "smooth", block: "center" });
    }

    onIntentConsumed();
  }, [scrollIntent, messages]);

  useEffect(() => {
    console.log(
      `[ChatMessages][${chatId}] hasMoreBottom changed: ${hasMoreBottom}, sentinel exists: ${!!bottomSentinelRef.current}`,
    );
  }, [hasMoreBottom]);

  // 3. Sentinel observer для подгрузки старых сообщений
  useEffect(() => {
    console.log(
      `[ChatMessages][${chatId}] topSentinel effect run, hasMore=${hasMore}, sentinel=${!!sentinelRef.current}`,
    );
    const sentinel = sentinelRef.current;
    if (!sentinel) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        console.log(
          `[ChatMessages][${chatId}] topSentinel intersecting=${entry.isIntersecting}`,
        );
        if (entry.isIntersecting) {
          prevScrollHeight.current = ref.current?.scrollHeight ?? null;
          onLoadMore();
        }
      },
      { threshold: 0.1 },
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [onLoadMore, hasMore]);

  useEffect(() => {
    console.log(
      `[ChatMessages][${chatId}] bottomSentinel effect run, hasMoreBottom=${hasMoreBottom}, sentinel=${!!bottomSentinelRef.current}`,
    );
    const sentinel = bottomSentinelRef.current;
    if (!sentinel) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        console.log(
          `[ChatMessages][${chatId}] bottomSentinel intersecting=${entry.isIntersecting}`,
        );
        if (entry.isIntersecting) onLoadMoreBottom();
      },
      { threshold: 0.1 },
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [onLoadMoreBottom, hasMoreBottom]);

  // 4. Восстановление позиции скролла после подгрузки — useLayoutEffect чтобы не было флика
  useLayoutEffect(() => {
    if (prevScrollHeight.current === null) return;
    const el = ref.current;
    if (!el) return;

    el.scrollTop = el.scrollHeight - prevScrollHeight.current;
    prevScrollHeight.current = null;
  }, [messages]);

  function handleScrollToMessage(id: string) {
    onScrollToMessage(id, messageRefs);
  }

  // Вычисляем firstUnreadId из intent один раз, чтобы рендерить разделитель

  return (
    <div ref={ref} style={styles.messageList}>
      {hasMore && <div ref={sentinelRef} style={{ height: 1 }} />}
      {loadingMore && <div style={styles.loadingMore}>Загрузка...</div>}

      {messages.map((m, index) => {
        const prev = messages[index - 1];
        const isGroupStart =
          !prev ||
          prev.senderId !== m.senderId ||
          !isSameDay(prev.createdAt, m.createdAt);

        return (
          <div key={m.id}>
            {m.id === firstUnreadIdRef.current && (
              <div style={styles.unreadDivider}>Новые сообщения</div>
            )}
            <MessageBubble
              message={{
                ...m,
                replyToMessage: m.replyToId
                  ? messages.find((x) => x.id === m.replyToId)
                  : undefined,
              }}
              user={getUser(m.senderId)}
              isGroupStart={isGroupStart}
              onImageClick={onImageClick}
              onReply={onReply}
              onForward={onForward}
              onEdit={onEdit}
              onDelete={onDelete}
              myId={myId}
              onScrollToMessage={handleScrollToMessage}
              highlight={m.id === highlightId}
              setRef={(el) => {
                const prev = messageRefs.current[m.id];
                if (prev && prev !== el) unobserveMessage(prev);
                if (!el && prev) unobserveMessage(prev);
                messageRefs.current[m.id] = el;
                if (el) observeMessage(el, m.id, m.createdAt);
              }}
              getUser={getUser}
            />
          </div>
        );
      })}
      {loadingMoreBottom && <div style={styles.loadingMore}>Загрузка...</div>}
      {hasMoreBottom && (
        <div
          ref={bottomSentinelRef}
          style={{ height: 1 }}
          data-debug="bottom-sentinel"
        />
      )}
    </div>
  );
}
