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
  observeMessage: (
    el: HTMLDivElement | null,
    id: string,
    createdAt: string,
  ) => void;
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
  firstUnreadId,
  observeMessage,
  hasMore, 
  loadingMore, 
  onLoadMore,
}: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const prevScrollHeight = useRef<number | null>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const messageRefs = useRef<Record<string, HTMLDivElement | null>>({});
  const scrolledToUnread = useRef(false);
  const unreadScrollDoneRef = useRef(false);

  useEffect(() => {
  if (loadingMore) {
    prevScrollHeight.current = ref.current?.scrollHeight ?? null;
  }
}, [loadingMore]);

const prevLoadingMore = useRef(false);
useEffect(() => {
  const wasLoading = prevLoadingMore.current;
  prevLoadingMore.current = loadingMore;

  if (wasLoading && !loadingMore && prevScrollHeight.current !== null) {
    const el = ref.current;
    if (!el) return;
    const diff = el.scrollHeight - prevScrollHeight.current;
    el.scrollTop += diff;
    prevScrollHeight.current = null;
  }
}, [loadingMore, messages]);

  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel) return;

    const observer = new IntersectionObserver(
      ([entry]) => { if (entry.isIntersecting) onLoadMore(); },
      { threshold: 0.1 },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [onLoadMore]);

  // Скролл к первому непрочитанному
  useEffect(() => {
    if (!firstUnreadId || scrolledToUnread.current) return;
    const el = messageRefs.current[firstUnreadId];
    if (!el) return;
    el.scrollIntoView({ behavior: "instant", block: "center" });
    scrolledToUnread.current = true;
    unreadScrollDoneRef.current = true; // скролл выполнен
    setTimeout(() => {}, 300);
  }, [messages, firstUnreadId]);

  useEffect(() => {
    scrolledToUnread.current = false;
    unreadScrollDoneRef.current = false;
  }, [chatId]);

  // Автоскролл вниз только если нет непрочитанных (обычный режим)
//   useEffect(() => {
//     if (!ref.current || firstUnreadId) return;
//     ref.current.scrollTop = ref.current.scrollHeight;
//   }, [messages]);

  //   useEffect(() => {
  //   console.log("firstUnreadId prop:", firstUnreadId);
  //   console.log("message ids in state:", messages.map(m => m.id));
  //   console.log("match found:", messages.some(m => m.id === firstUnreadId));
  // }, [firstUnreadId, messages]);

  function handleScrollToMessage(id: string) {
    onScrollToMessage(id, messageRefs);
  }

  return (
    <div ref={ref} style={styles.messageList}>
		{hasMore && <div ref={sentinelRef} style={{ height: 1 }} />}
		{loadingMore && (
        <div style={styles.loadingMore}>Загрузка...</div>
      )}
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
                messageRefs.current[m.id] = el;
                observeMessage(el, m.id, m.createdAt);
              }}
              getUser={getUser}
            />
          </div>
        );
      })}
    </div>
  );
}
