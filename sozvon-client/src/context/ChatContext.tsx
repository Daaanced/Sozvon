// sozvon-client/src/context/ChatContext.tsx
import { createContext, useContext, useEffect, useState, useRef } from "react";
import { getUserById, searchUser, User } from "../api/users";
import { onWSMessage } from "../services/ws";
import { requestAuth } from "../api/http";

type Chat = {
  chatId: string;
  type: string;
  name?: string;
  members: number[];
  lastMessage?: string;
  updatedAt?: string;
};

type ChatContextType = {
  chats: Chat[];
  users: Record<number, User>;
  myId: number;
  myLogin: string;
  me: User | null;
  unread: Record<string, boolean>;
  markRead: (chatId: string, lastMessageId?: string) => void;
  notifyOwnMessage: (chatId: string, createdAt: string) => void;
  getSafeUser: (id: number) => User;
};

export const DELETED_USER: User = {
  id: 0,
  login: "-",
  name: "Deleted",
  email: "-",
  info: "-",
  picture: "http://92.127.177.190:8080/static/avatars/deleted.png",
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
  if (!token) {
    console.error("ChatProvider: no token");
    return null;
  }

  const parsed = parseTokenPayload(token);
  if (!parsed) {
    console.error("ChatProvider: invalid token");
    return null;
  }

  const { id: myId, login: myLogin } = parsed;

  const [me, setMe] = useState<User | null>(null);
  const [chats, setChats] = useState<Chat[]>([]);
  const [users, setUsers] = useState<Record<number, User>>({});
  const [unread, setUnread] = useState<Record<string, boolean>>({});
  const loadingUsersRef = useRef<Set<number>>(new Set());

  function getSafeUser(id: number): User {
    if (id === myId) return me ?? DELETED_USER;
    return users[id] ?? DELETED_USER;
  }

  async function markRead(chatId: string, lastMessageId?: string) {
    console.log("[markRead]", chatId, lastMessageId);
    setUnread((prev) => ({ ...prev, [chatId]: false }));
    if (!lastMessageId) return;
    try {
      await requestAuth(`/chats/${chatId}/read`, {
        method: "POST",
        body: JSON.stringify({ lastMessageId }),
      });
    } catch (e) {
      console.error("markRead failed:", e);
    }
  }

  function notifyOwnMessage(chatId: string, createdAt: string) {
    console.log(
      `[notifyOwnMessage] chatId=${chatId.slice(0, 8)} createdAt=${createdAt}`,
    );
    setChats((prev) => {
      const index = prev.findIndex((c) => c.chatId === chatId);
      if (index === -1) return prev;

      const chat = prev[index];
      const prevTime = chat.updatedAt ? new Date(chat.updatedAt).getTime() : 0;
      const newTime = new Date(createdAt).getTime();
      if (newTime < prevTime) return prev;

      const updatedChat = { ...chat, updatedAt: createdAt };
      const next = [...prev];
      next.splice(index, 1);
      next.unshift(updatedChat);
      return next;
    });
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
      const data = await requestAuth("/chats");
      const safeChats: Chat[] = Array.isArray(data) ? data : [];
      const sorted = sortChats(safeChats);

      console.log(
        "[loadChats] from server:",
        sorted.map((c) => ({
          id: c.chatId.slice(0, 8),
          updatedAt: c.updatedAt,
        })),
      );

      setChats((prev) => {
        if (prev.length === 0) return sorted;

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
          const winner = prevTime > newTime ? existing : chat;
          console.log(
            `[loadChats] merge ${chat.chatId.slice(0, 8)}: server=${chat.updatedAt} client=${existing.updatedAt} → kept=${winner.updatedAt}`,
          );
          return winner;
        });

        const result = sortChats(merged);
        console.log(
          "[loadChats] final order:",
          result.map((c) => ({
            id: c.chatId.slice(0, 8),
            updatedAt: c.updatedAt,
          })),
        );
        return result;
      });

      // Обновляем unread только для чатов где ещё не сбросили локально
      setUnread((prev) => {
        const next = { ...prev };
        sorted.forEach((chat: any) => {
          if (chat.unreadCount > 0 && prev[chat.chatId] !== false) {
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
        console.log("[WS] chat event:", msg.event, msg.data);
        loadChats();
        return;
      }

      if (msg.event === "message:new") {
        const chatId: string | undefined = msg.data?.chatId;
        if (!chatId) return;

        setChats((prev) => {
          const index = prev.findIndex((c) => c.chatId === chatId);
          if (index === -1) return prev;

          const chat = prev[index];
          const now = msg.data.createdAt ?? new Date().toISOString();
          console.log(
            `[WS message:new] chatId=${chatId.slice(0, 8)} prevUpdatedAt=${chat.updatedAt} newTime=${now}`,
          );
          const prevTime = chat.updatedAt
            ? new Date(chat.updatedAt).getTime()
            : 0;
          const newTime = new Date(now).getTime();

          if (newTime < prevTime) return prev;

          const updatedChat = { ...chat, updatedAt: now };
          const next = [...prev];
          next.splice(index, 1);
          next.unshift(updatedChat);
          console.log(
            "[WS setChats] new order:",
            next.map(
              (c) => `${c.chatId.slice(0, 8)}=${c.updatedAt?.slice(11, 19)}`,
            ),
          );
          return next;
        });

        setUnread((prev) => {
          if (prev[chatId]) return prev;
          return { ...prev, [chatId]: true };
        });
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
        markRead,
        notifyOwnMessage,
        getSafeUser,
      }}
    >
      {children}
    </ChatContext.Provider>
  );
}
