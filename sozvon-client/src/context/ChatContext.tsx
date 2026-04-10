// sozvon-client/src/context/ChatContext.tsx
import { createContext, useContext, useEffect, useState, useRef } from "react";
import { getUserById, searchUser, User } from "../api/users";
import { onWSMessage } from "../services/ws";
import { getChats, markRead } from "../api/chats";

type Chat = {
  chatId: string;
  type: string;
  name?: string;
  members: number[];
  lastMessageId?: string;
  lastReadMessageId?: string;
  updatedAt?: string;
};

type ChatContextType = {
  chats: Chat[];
  users: Record<number, User>;
  myId: number;
  myLogin: string;
  me: User | null;
  unread: Record<string, boolean>;
  loadUser: (id: number) => void;
  handleMarkRead: (chatId: string, lastMessageId?: string) => void;
  notifyOwnMessage: (
    chatId: string,
    createdAt: string,
    messageId?: string,
  ) => void;
  getSafeUser: (id: number) => User;
  setActiveChat: (chatId: string | null) => void;
};

export const DELETED_USER: User = {
  id: 0,
  login: "-",
  name: "Deleted",
  email: "-",
  info: "-",
  picture: "http://92.127.169.188:8080/static/avatars/deleted.png",
  created_at: "-",
  updated_at: "-",
};

const ChatContext = createContext<ChatContextType | null>(null);

export function useChatContext() {
  const ctx = useContext(ChatContext);
  if (!ctx) throw new Error("useChatContext must be used inside ChatProvider");
  return ctx;
}

function parseTokenPayload(
  token: string,
): { id: number; login: string } | null {
  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    const id = parseInt(payload.user_id, 10);
    const login = payload.login ?? "";
    if (!id) return null;
    return { id, login };
  } catch {
    return null;
  }
}

function sortChats(chats: Chat[]): Chat[] {
  return [...chats].sort((a, b) => {
    const ta = a.updatedAt ? new Date(a.updatedAt).getTime() : 0;
    const tb = b.updatedAt ? new Date(b.updatedAt).getTime() : 0;
    return tb - ta;
  });
}

export function ChatProvider({ children }: { children: React.ReactNode }) {
  const token = localStorage.getItem("token");
  if (!token) return null;

  const parsed = parseTokenPayload(token);
  if (!parsed) return null;

  const { id: myId, login: myLogin } = parsed;

  const [me, setMe] = useState<User | null>(null);
  const [chats, setChats] = useState<Chat[]>([]);
  const [users, setUsers] = useState<Record<number, User>>({});
  const [unread, setUnread] = useState<Record<string, boolean>>({});

  const loadingUsersRef = useRef<Set<number>>(new Set());
  const activeChatRef = useRef<string | null>(null);

  const chatsRef = useRef<Chat[]>([]);

  function setActiveChat(chatId: string | null) {
	const prev = activeChatRef.current;
	activeChatRef.current = chatId;

	// При уходе из чата — пересчитываем unread для предыдущего
	if (prev && prev !== chatId) {
		recheckUnread(prev);
	}

	if (!chatId) return;
	setUnread((prev) => ({ ...prev, [chatId]: false }));
	}

  function getSafeUser(id: number): User {
    if (id === myId) return me ?? DELETED_USER;
    return users[id] ?? DELETED_USER;
  }

	function recheckUnread(chatId: string) {
		const chat = chatsRef.current.find((c) => c.chatId === chatId);
		if (!chat?.lastMessageId) return;

		const isRead = chat.lastReadMessageId === chat.lastMessageId;
		setUnread((prev) => ({ ...prev, [chatId]: !isRead }));
	}

	async function handleMarkRead(chatId: string, lastMessageId?: string) {
	if (lastMessageId) {
		// Обновляем lastReadMessageId в локальном state чата
		setChats((prev) => {
		const next = prev.map((c) =>
			c.chatId === chatId
			? { ...c, lastReadMessageId: lastMessageId }
			: c
		);
		chatsRef.current = next;
		return next;
		});
	}

	if (chatId === activeChatRef.current) {
		setUnread((prev) => ({ ...prev, [chatId]: false }));
	} else {
		recheckUnread(chatId);
	}

	if (!lastMessageId) return;
	try {
		await markRead(chatId, lastMessageId);
	} catch (e) {
		console.error("markRead failed:", e);
	}
	}

  function notifyOwnMessage(chatId: string, createdAt: string, messageId?: string) {
	setChats((prev) => {
		const index = prev.findIndex((c) => c.chatId === chatId);
		if (index === -1) return prev;

		const chat = prev[index];
		const updatedChat = {
		...chat,
		updatedAt: createdAt,
		...(messageId ? { lastMessageId: messageId, lastReadMessageId: messageId } : {}),
		};
		const next = [...prev];
		next.splice(index, 1);
		next.unshift(updatedChat);
		chatsRef.current = next;
		return next;
	});

	setUnread((prev) => ({ ...prev, [chatId]: false }));
	}

  function loadUserById(id: number) {
    if (id === myId) return;
    if (users[id] || loadingUsersRef.current.has(id)) return;

    loadingUsersRef.current.add(id);
    getUserById(id)
      .then((u) => setUsers((p) => ({ ...p, [id]: u ?? DELETED_USER })))
      .catch(() => setUsers((p) => ({ ...p, [id]: DELETED_USER })))
      .finally(() => loadingUsersRef.current.delete(id));
  }

  async function loadChats() {
    try {
      const data = await getChats();
      const safeChats: Chat[] = Array.isArray(data) ? data : [];
      const sorted = sortChats(safeChats);

      setChats((prev) => {
        if (prev.length === 0) {
          chatsRef.current = sorted;
          return sorted;
        }

        const map = new Map(prev.map((c) => [c.chatId, c]));
        const merged = sorted.map((chat) => {
          const existing = map.get(chat.chatId);
          if (!existing) return chat;

          const prevTime = existing.updatedAt
            ? new Date(existing.updatedAt).getTime()
            : 0;
          const newTime = chat.updatedAt
            ? new Date(chat.updatedAt).getTime()
            : 0;

          return prevTime > newTime ? existing : chat;
        });

        const result = sortChats(merged);
        chatsRef.current = result;
        return result;
      });

      setUnread((prev) => {
		const next = { ...prev };
		sorted.forEach((chat: any) => {
			if (chat.chatId === activeChatRef.current) return; // открытый чат — не трогаем

			const isRead =
			!chat.lastMessageId ||
			chat.lastReadMessageId === chat.lastMessageId;

			if (!isRead) {
			next[chat.chatId] = true;
			}
		});
		return next;
		});

      sorted.forEach((chat) => {
        chat.members.filter((id: number) => id !== myId).forEach(loadUserById);
      });
    } catch {
      setChats([]);
    }
  }

  useEffect(() => {
    let mounted = true;

    searchUser(myLogin)
      .then((u) => {
        if (mounted) setMe(u);
      })
      .catch(() => {});

    loadChats();

    const off = onWSMessage((msg) => {
      if (!mounted) return;

      if (msg.event === "chat:created" || msg.event === "chat:activated") {
        loadChats();
        return;
      }

      if (msg.event === "message:new") {
		const chatId: string | undefined = msg.data?.chatId;
		const senderId: number | undefined = msg.data?.senderId;
		if (!chatId) return;
		if (senderId === myId) return;

		const messageId: string | undefined = msg.data?.messageId ?? msg.data?.id;

		setChats((prev) => {
			const index = prev.findIndex((c) => c.chatId === chatId);
			if (index === -1) return prev;
			const chat = prev[index];
			const now = msg.data.createdAt ?? new Date().toISOString();
			const prevTime = chat.updatedAt ? new Date(chat.updatedAt).getTime() : 0;
			const newTime = new Date(now).getTime();
			if (newTime < prevTime) return prev;

			const updatedChat = {
			...chat,
			updatedAt: now,
			...(messageId ? { lastMessageId: messageId } : {}),
			// lastReadMessageId НЕ меняем — чужое сообщение мы не читали
			};
			const next = [...prev];
			next.splice(index, 1);
			next.unshift(updatedChat);
			chatsRef.current = next;
			return next;
		});

		if (chatId === activeChatRef.current) return; // чат открыт — точка не нужна

		setUnread((prev) => ({ ...prev, [chatId]: true }));
		}
    });

    return () => {
      mounted = false;
      off();
    };
  }, []);

  return (
    <ChatContext.Provider
      value={{
        chats,
        users,
        myId,
        myLogin,
        me,
        unread,
        handleMarkRead,
        notifyOwnMessage,
        getSafeUser,
        setActiveChat,
        loadUser: loadUserById,
      }}
    >
      {children}
    </ChatContext.Provider>
  );
}
