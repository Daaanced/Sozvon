// sozvon-client/src/components/chat/ChatMessages.tsx

import { useRef, useEffect, useLayoutEffect, useState } from "react";
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
  onJumpToBottom: () => void;
  initialized: boolean;
  onBottomSentinelHidden: () => void;
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
  onJumpToBottom,
  initialized,
  onBottomSentinelHidden,
}: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const bottomSentinelRef = useRef<HTMLDivElement>(null);
  const messageRefs = useRef<Record<string, HTMLDivElement | null>>({});
  const prevScrollHeight = useRef<number | null>(null);
  const firstUnreadIdRef = useRef<string | null>(null);
  const isAtBottomRef = useRef(true);
  const [showScrollBtn, setShowScrollBtn] = useState(false);
  const prevMessageCountRef = useRef(0);

  if (scrollIntent?.type === "unread" && !firstUnreadIdRef.current) {
    firstUnreadIdRef.current = scrollIntent.id;
  }

  // 1. Сброс при смене чата
  useEffect(() => {
    console.log(`[ChatMessages][${chatId}] RESET on chatId change`);
    messageRefs.current = {};
    firstUnreadIdRef.current = null;
    prevMessageCountRef.current = 0;
    isAtBottomRef.current = true;
    setShowScrollBtn(false);
    if (ref.current) ref.current.scrollTop = 0;
  }, [chatId]);

  // 2. Применение scroll intent — useLayoutEffect чтобы DOM уже содержал новые сообщения
  useLayoutEffect(() => {
    if (!scrollIntent || messages.length === 0) return;

    const el = ref.current;
    if (!el) return;

    prevScrollHeight.current = null;

    if (scrollIntent.type === "bottom") {
      el.scrollTop = el.scrollHeight;
      isAtBottomRef.current = true;
      setShowScrollBtn(false);
      onIntentConsumed();
      return;
    }

    const target = messageRefs.current[scrollIntent.id];
    if (!target) return; // сообщение ещё не в DOM — ждём следующего рендера

    if (scrollIntent.type === "unread") {
      target.scrollIntoView({ behavior: "instant", block: "center" });
      setTimeout(() => {
        onBottomSentinelHidden();
        onIntentConsumed();

        // Принудительно проверяем видимость bottomSentinel
        const sentinel = bottomSentinelRef.current;
        const container = ref.current;
        if (sentinel && container) {
          const sr = sentinel.getBoundingClientRect();
          const cr = container.getBoundingClientRect();
          if (sr.top < cr.bottom && sr.bottom > cr.top) {
            onLoadMoreBottom();
          }
        }
      }, 300);
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
        if (entry.isIntersecting) {
          onLoadMoreBottom();
        }
        //   else {
        //   console.log(`[ChatMessages][${chatId}] bottomSentinel hidden → clearScrollingToUnread`);
        // }
      },
      { threshold: 0.1 },
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [onLoadMoreBottom, hasMoreBottom]);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    const handleScroll = () => {
      const threshold = 80;
      const atBottom =
        el.scrollHeight - el.scrollTop - el.clientHeight < threshold;

      isAtBottomRef.current = atBottom;

      setShowScrollBtn(el.scrollHeight > el.clientHeight && !atBottom);
    };

    el.addEventListener("scroll", handleScroll, { passive: true });
    return () => el.removeEventListener("scroll", handleScroll);
  }, []);

  // 4. Восстановление позиции скролла после подгрузки — useLayoutEffect чтобы не было флика
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;

    // Приоритет 1: восстановление позиции после подгрузки вверх
    if (prevScrollHeight.current !== null) {
      el.scrollTop = el.scrollHeight - prevScrollHeight.current;
      prevScrollHeight.current = null;
      prevMessageCountRef.current = messages.length; // синхронизируем счётчик
      return;
    }

    // Приоритет 2: авто-скролл вниз при новых сообщениях
    const newCount = messages.length;
    const prevCount = prevMessageCountRef.current;
    prevMessageCountRef.current = newCount;

    if (newCount > prevCount && prevCount > 0 && isAtBottomRef.current) {
      el.scrollTop = el.scrollHeight;
    }
  }, [messages]);

  function handleScrollToMessage(id: string) {
    onScrollToMessage(id, messageRefs);
  }
  // Вычисляем firstUnreadId из intent один раз, чтобы рендерить разделитель

  return (
    <div
      style={{
        position: "relative",
        flex: 1,
        display: "flex",
        flexDirection: "column",
        minHeight: 0,
      }}
    >
      <div ref={ref} style={styles.messageList}>
        <div style={{ flex: 1 }} />
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
        {initialized && hasMoreBottom && (
          <div
            ref={bottomSentinelRef}
            style={{ height: 1 }}
            data-debug="bottom-sentinel"
          />
        )}
      </div>
      {(showScrollBtn || hasMoreBottom) && (
        <button onClick={onJumpToBottom} style={styles.scrollToBottomBtn}>
          ↓
        </button>
      )}
    </div>
  );
}
