// sozvon-client/src/components/chat/Chat.tsx

import { useState, useEffect } from "react";
import { useChatMessages } from "./hooks/useChatMessages";
import { useChatActions } from "./hooks/useChatActions";
import { useFileUpload } from "./hooks/useFileUpload";
import { useChatContext } from "../../context/ChatContext";
import { useVisibleMessages } from "./hooks/useVisibleMessages";
import ChatMessages from "./ChatMessages";
import ChatInput from "./ChatInput";
import ChatTopBar from "./ChatTopBar";
import Lightbox from "./Lightbox";
import ForwardModal from "./ForwardModal";
import EditModal from "./EditModal";
import { styles } from "./chat.styles";
import { Message } from "./chat.types";

type Props = { chatId: string };

export default function Chat({ chatId }: Props) {
  const { getSafeUser, chats, myId, handleMarkRead, setActiveChat } =
    useChatContext();

  const [lightboxUrl, setLightboxUrl] = useState<string | null>(null);
  const [forwardingMessage, setForwardingMessage] = useState<Message | null>(
    null,
  );

  const { observe, unobserve, flush, reset } =
	useVisibleMessages((id) => {
		handleMarkRead(chatId, id);
	});

  const {
    messages,
    setMessages,
    scrollIntent,
    consumeScrollIntent,
    highlightId,
    hasMore,
    loadingMore,
    loadMore,
    hasMoreBottom,
    loadingMoreBottom,
    loadMoreBottom,
    scrollToMessage,
    jumpToBottom,
    initialized,
    clearScrollingToUnread,
  } = useChatMessages(chatId);

  const {
    replyTo,
    setReplyTo,
    editingMessage,
    editText,
    setEditText,
    handleEdit,
    submitEdit,
    cancelEdit,
    handleDelete,
    send,
  } = useChatActions(chatId, setMessages);

  const {
    pendingFiles,
    uploading,
    uploadProgress,
    handleFilesAdded,
    handleFileRemove,
    clearFiles,
  } = useFileUpload();

  const [text, setText] = useState("");

  const chat = chats.find((c) => c.chatId === chatId);
  const withId = chat?.members.find((m) => m !== myId);
  const companion = withId ? getSafeUser(withId) : null;

  useEffect(() => {
    reset();
    setActiveChat(chatId);

    return () => {
      flush();
      setActiveChat(null);
    };
  }, [chatId]);

  // Логируем каждое изменение scrollIntent
  useEffect(() => {
    console.log(
      `[Chat][${chatId}] scrollIntent changed:`,
      JSON.stringify(scrollIntent),
    );
  }, [scrollIntent]);

  // Логируем изменение messages
  useEffect(() => {
    if (messages.length === 0) return;
    //первое и последнее загруженное сообщение
    const first = messages[0];
    const last = messages[messages.length - 1];
    console.log(
      `[Chat][${chatId}] messages updated: count=${messages.length}, ` +
        `first="${first.id}"(${first.createdAt}), last="${last.id}"(${last.createdAt}), hasMore=${hasMore}`,
    );
  }, [messages]);

  function handleSend() {
    send({
      text,
      pendingFiles,
      replyTo,
      onClear: () => {
        setText("");
        clearFiles();
      },
    });

    if (hasMoreBottom) {
      setTimeout(() => jumpToBottom(), 300); // даём WS время вернуть новое сообщение
    } else {
      jumpToBottom();
    }
  }

  return (
    <div style={styles.chatWrapper}>
      {lightboxUrl && (
        <Lightbox url={lightboxUrl} onClose={() => setLightboxUrl(null)} />
      )}

      {forwardingMessage && (
        <ForwardModal
          message={forwardingMessage}
          onClose={() => setForwardingMessage(null)}
        />
      )}

      {editingMessage && (
        <EditModal
          value={editText}
          onChange={setEditText}
          onSave={submitEdit}
          onCancel={cancelEdit}
        />
      )}

      <ChatTopBar
        user={companion}
        onCall={() => console.log("call", chatId)}
        onSettings={() => console.log("settings", chatId)}
      />

      <ChatMessages
        chatId={chatId}
        messages={messages}
        getUser={getSafeUser}
        onImageClick={setLightboxUrl}
        onReply={setReplyTo}
        onForward={setForwardingMessage}
        onEdit={handleEdit}
        onDelete={handleDelete}
        myId={myId}
        highlightId={highlightId}
        onScrollToMessage={scrollToMessage}
        scrollIntent={scrollIntent}
        onIntentConsumed={consumeScrollIntent}
        observeMessage={observe}
        unobserveMessage={unobserve}
        hasMore={hasMore}
        loadingMore={loadingMore}
        onLoadMore={loadMore}
        hasMoreBottom={hasMoreBottom}
        loadingMoreBottom={loadingMoreBottom}
        onLoadMoreBottom={loadMoreBottom}
        onJumpToBottom={jumpToBottom}
        initialized={initialized}
        onBottomSentinelHidden={clearScrollingToUnread}
      />

      <ChatInput
        text={text}
        pendingFiles={pendingFiles}
        uploading={uploading}
        uploadProgress={uploadProgress}
        replyTo={replyTo}
        onCancelReply={() => setReplyTo(null)}
        onTextChange={setText}
        onSend={handleSend}
        onFilesAdded={handleFilesAdded}
        onFileRemove={handleFileRemove}
      />
    </div>
  );
}
