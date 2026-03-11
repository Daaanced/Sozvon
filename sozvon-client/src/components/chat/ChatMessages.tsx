//sozvon-client\src\components\chat\ChatMessages.tsx
import { useRef, useEffect } from "react";
import MessageBubble from "./MessageBubble";
import { styles } from "./chat.styles";
import type { Message } from "./chat.types";
import type { User } from "../../api/users";

type Props = {
  messages: Message[];
  getUser: (senderId: number) => User;
  onImageClick: (url: string) => void;
  onReply: (message: Message) => void;
  onForward: (message: Message) => void;
  highlightId: string | null;
  onScrollToMessage: (
    id: string,
    messageRefs: React.MutableRefObject<Record<string, HTMLDivElement | null>>,
  ) => void;
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
  messages,
  getUser,
  onImageClick,
  onReply,
  onForward,
  highlightId,
  onScrollToMessage,
}: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const messageRefs = useRef<Record<string, HTMLDivElement | null>>({});

  // Автоскролл вниз при новых сообщениях
  useEffect(() => {
    if (ref.current) {
      ref.current.scrollTop = ref.current.scrollHeight;
    }
  }, [messages]);

  // Заменить плашку reply в map на клик с передачей рефов:
  function handleScrollToMessage(id: string) {
    onScrollToMessage(id, messageRefs);
  }

  return (
    <div ref={ref} style={styles.messageList}>
      {messages.map((m, index) => {
        const prev = messages[index - 1];
        const isGroupStart =
          !prev ||
          prev.senderId !== m.senderId ||
          !isSameDay(prev.createdAt, m.createdAt);
        const replyToMessage = m.replyToId
          ? messages.find((x) => x.id === m.replyToId)
          : undefined;
        return (
          <MessageBubble
            key={m.id}
            message={{ ...m, replyToMessage }}
            user={getUser(m.senderId)}
            isGroupStart={isGroupStart}
            onImageClick={onImageClick}
            onReply={onReply}
            onForward={onForward}
            onScrollToMessage={handleScrollToMessage}
            highlight={m.id === highlightId}
            setRef={(el) => {
              messageRefs.current[m.id] = el;
            }}
          />
        );
      })}
    </div>
  );
}
