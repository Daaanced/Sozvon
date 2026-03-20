// sozvon-client/src/components/chat/Chat.tsx

import { useState, useEffect, useRef } from "react";
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
  const { getSafeUser, chats, myId, markRead, setActiveChat } = useChatContext();

  const [lightboxUrl, setLightboxUrl] = useState<string | null>(null);
  const [forwardingMessage, setForwardingMessage] = useState<Message | null>(
    null,
  );

  const { messages, setMessages, firstUnreadId, highlightId,
  hasMore, loadingMore, loadMore, scrollToMessage, initialLoading } = useChatMessages(chatId);

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
  } = useFileUpload();

  function handleSend() {
    send({ text, pendingFiles, replyTo, onClear: () => setText("") });
  }

  const [text, setText] = useState("");

  const chat = chats.find((c) => c.chatId === chatId);
  const withId = chat?.members.find((m) => m !== myId);
  const companion = withId ? getSafeUser(withId) : null;
  const chatIdRef = useRef(chatId);

  useEffect(() => {
  chatIdRef.current = chatId;
  setActiveChat(chatId);
  return () => setActiveChat(null);
}, [chatId]);

  const { observe, unobserve, flush, reset, initChat } = useVisibleMessages(
  (id) => markRead(chatId, id),
);

	useEffect(() => {
	if (messages.length === 0) return;
	if (firstUnreadId) return; // если есть unread — позиция уже правильная на сервере

	const lastMsg = messages[messages.length - 1];
	initChat(chatId, new Date(lastMsg.createdAt).getTime());
	}, [chatId, messages, firstUnreadId]);

	useEffect(() => {
	const currentChatId = chatId;
	return () => {
		flush(currentChatId);
	};
	}, [chatId, flush]);

	useEffect(() => {
	reset();
	}, [chatId, reset]);

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
        firstUnreadId={firstUnreadId}
        observeMessage={observe}
		unobserveMessage={unobserve}
		hasMore={hasMore}
		loadingMore={loadingMore}
		onLoadMore={loadMore}
		initialLoading={initialLoading}
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
