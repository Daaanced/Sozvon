//sozvon-client\src\components\chat\ChatMessages.tsx

import { useRef, useEffect } from "react";
import MessageBubble from "./MessageBubble";
import { styles } from "./chat.styles";
import type { Message } from "./chat.types";
import type { User } from "../../api/users";

type Props = {
  chatId: string;
  messages: Message[];
  getUser: (senderId: number) => User;
  onImageClick: (url: string) => void;
  onReply: (message: Message) => void;
  onForward: (message: Message) => void;
  onEdit: (message: Message) => void;
  onDelete: (message: Message) => void;
  observeMessage: (el: HTMLDivElement | null, id: string, createdAt: string) => void;
  unobserveMessage: (el: HTMLDivElement | null) => void;
  myId: number;
  highlightId: string | null;
  onScrollToMessage: (
    id: string,
    messageRefs: React.MutableRefObject<Record<string, HTMLDivElement | null>>,
  ) => void;
  firstUnreadId: string | null;
  hasMore: boolean;
  loadingMore: boolean;
  onLoadMore: () => void;
  initialLoading: boolean;
};

function isSameDay(a: string, b: string) {
  const d1 = new Date(a), d2 = new Date(b);
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
  firstUnreadId,
  observeMessage,
  unobserveMessage,
  hasMore,
  loadingMore,
  onLoadMore,
  initialLoading,
}: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const messageRefs = useRef<Record<string, HTMLDivElement | null>>({});

  const prevScrollHeight = useRef<number | null>(null);
  const prevLoadingMore = useRef(false);
  const scrolledToUnread = useRef(false);
  const initialScrollDoneRef = useRef(false);

  // 1. Сброс всего при смене чата
  useEffect(() => {
    console.log(`[ChatMessages] chatId changed, resetting`);
    scrolledToUnread.current = false;
    initialScrollDoneRef.current = false;
    prevScrollHeight.current = null;
    if (ref.current) {
      ref.current.scrollTop = 0;
    }
  }, [chatId]);

  // 2. Sentinel observer для подгрузки старых сообщений
  useEffect(() => {
  if (initialLoading) return; // не создаём observer пока грузится
  const sentinel = sentinelRef.current;
  if (!sentinel) return;

  console.log(`[ChatMessages] sentinel observer created`);

  const observer = new IntersectionObserver(
    ([entry]) => {
      console.log(`[ChatMessages] sentinel intersecting=${entry.isIntersecting}`);
      if (entry.isIntersecting) {
        prevScrollHeight.current = ref.current?.scrollHeight ?? null;
        onLoadMore();
      }
    },
    { threshold: 0.1 },
  );
  observer.observe(sentinel);
  return () => observer.disconnect();
}, [onLoadMore, initialLoading]);

  // 3. Ресторация позиции скролла после подгрузки старых сообщений
  useEffect(() => {
    const wasLoading = prevLoadingMore.current;
    prevLoadingMore.current = loadingMore;

    if (wasLoading && !loadingMore && prevScrollHeight.current !== null) {
      const el = ref.current;
      if (!el) return;

      const diff = el.scrollHeight - prevScrollHeight.current;
      console.log(`[ChatMessages] loadMore scroll restore: diff=${diff}, scrollTopBefore=${el.scrollTop}`);
      el.scrollTop += diff;
      prevScrollHeight.current = null;
      console.log(`[ChatMessages] scrollTop after restore: ${el.scrollTop}`);
    }
  }, [loadingMore, messages]);

  // 4. Скролл к первому непрочитанному
  useEffect(() => {
    if (!firstUnreadId || scrolledToUnread.current) return;
    const el = messageRefs.current[firstUnreadId];
    if (!el) return;
    console.log(`[ChatMessages] Scrolling to firstUnreadId="${firstUnreadId}"`);
    el.scrollIntoView({ behavior: "instant", block: "center" });
    scrolledToUnread.current = true;
  }, [messages, firstUnreadId]);

  // 5. Скролл вниз при первой загрузке без unread
  useEffect(() => {
  if (initialScrollDoneRef.current) return;
  if (messages.length === 0) return;
  if (firstUnreadId) return;

  const el = ref.current;
  if (!el) return;

  el.scrollTop = el.scrollHeight;
  initialScrollDoneRef.current = true;
  console.log(`[ChatMessages] Scrolled to bottom: scrollTop=${el.scrollTop}, scrollHeight=${el.scrollHeight}`);

  // Проверяем через 500мс не изменилась ли высота
  const t1 = setTimeout(() => {
    console.log(`[ChatMessages] 500ms later: scrollTop=${el.scrollTop}, scrollHeight=${el.scrollHeight}, isAtBottom=${el.scrollTop + el.clientHeight >= el.scrollHeight - 5}`);
  }, 500);
  const t2 = setTimeout(() => {
    console.log(`[ChatMessages] 1500ms later: scrollTop=${el.scrollTop}, scrollHeight=${el.scrollHeight}, isAtBottom=${el.scrollTop + el.clientHeight >= el.scrollHeight - 5}`);
  }, 1500);

  return () => { clearTimeout(t1); clearTimeout(t2); };
}, [messages, firstUnreadId]);

  // Лог для отладки
  useEffect(() => {
    if (messages.length > 0) {
      const el = ref.current;
      console.log(`[ChatMessages] After render: scrollTop=${el?.scrollTop}, scrollHeight=${el?.scrollHeight}, clientHeight=${el?.clientHeight}`);
    }
  }, [messages]);

  function handleScrollToMessage(id: string) {
    onScrollToMessage(id, messageRefs);
  }

  return (
    <div ref={ref} style={styles.messageList}>
      {hasMore && !initialLoading && <div ref={sentinelRef} style={{ height: 1 }} />}
      {loadingMore && <div style={styles.loadingMore}>Загрузка...</div>}

      {messages.map((m, index) => {
        const prev = messages[index - 1];
        const isGroupStart =
          !prev ||
          prev.senderId !== m.senderId ||
          !isSameDay(prev.createdAt, m.createdAt);

        return (
          <div key={m.id}>
            {m.id === firstUnreadId && (
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
    </div>
  );
}