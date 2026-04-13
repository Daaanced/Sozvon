// sozvon-client/src/api/chats.ts
import { requestAuth } from "./http";
import { PendingFile } from "../components/chat/chat.types";

const API_URL = "http://92.127.169.188:8080";

export function createChat(toUserId: number) {
  return requestAuth("/chats/create", {
    method: "POST",
    body: JSON.stringify({ to_id: toUserId }),
  });
}

export function createGroupChat(name: string, memberIds: number[]) {
  return requestAuth("/chats/create", {
    method: "POST",
    body: JSON.stringify({ name, member_ids: memberIds }),
  });
}

export function getChats() {
  return requestAuth("/chats");
}

export function getMessages(chatId: string, limit = 50, offset = 0) {
  return requestAuth(
    `/chats/${chatId}/messages?limit=${limit}&offset=${offset}`,
  );
}

export function getMessagesContext(chatId: string, messageId: string) {
  return requestAuth(`/chats/${chatId}/messages/${messageId}/context`);
}

export function sendMessage(chatId: string, text: string, replyToId?: string) {
  return requestAuth(`/chats/${chatId}/messages`, {
    method: "POST",
    body: JSON.stringify({ text, replyToId }),
  });
}

// Отправка с файлами через multipart/form-data
export function uploadFiles(
  chatId: string,
  text: string,
  pendingFiles: PendingFile[],
  replyToId?: string,
  onProgress?: (percent: number) => void,
): Promise<any> {
  return new Promise((resolve, reject) => {
    const token = localStorage.getItem("token");
    const formData = new FormData();

    if (text.trim()) {
      formData.append("text", text.trim());
    }

    if (replyToId) formData.append("replyToId", replyToId);

    // файлы
    pendingFiles.forEach((pf) => formData.append("files", pf.file));

    // мета
    formData.append(
      "meta",
      JSON.stringify(
        pendingFiles.map((pf) => ({
          width: pf.width,
          height: pf.height,
        })),
      ),
    );

    const xhr = new XMLHttpRequest();

    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable && onProgress) {
        onProgress(Math.round((e.loaded / e.total) * 100));
      }
    };

    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          resolve(JSON.parse(xhr.responseText));
        } catch {
          reject(new Error("Invalid response"));
        }
      } else {
        reject(new Error(xhr.responseText || "Upload failed"));
      }
    };

    xhr.onerror = () => reject(new Error("Network error"));

    xhr.open("POST", `${API_URL}/chats/${chatId}/upload`);
    if (token) xhr.setRequestHeader("Authorization", `Bearer ${token}`);
    xhr.send(formData);
  });
}

export function forwardMessages(
  messageIds: string[],
  toChatId: string,
  commentText?: string,
) {
  return requestAuth("/chats/forward", {
    method: "POST",
    body: JSON.stringify({ messageIds, toChatId, commentText }),
  });
}

export function editMessage(chatId: string, messageId: string, text: string) {
  return requestAuth(`/chats/${chatId}/messages/${messageId}`, {
    method: "PUT",
    body: JSON.stringify({ text }),
  });
}

export function deleteMessage(chatId: string, messageId: string) {
  return requestAuth(`/chats/${chatId}/messages/${messageId}`, {
    method: "DELETE",
  });
}

export function markRead(chatId: string, lastMessageId: string) {
  return requestAuth(`/chats/${chatId}/read`, {
    method: "POST",
    body: JSON.stringify({ lastMessageId }),
  });
}

export function getChatInfo(chatId: string) {
  return requestAuth(`/chats/${chatId}`);
}

export function getUnreadMessages(chatId: string) {
  return requestAuth(`/chats/${chatId}/messages/unread`);
}

export function getMessagesAfter(
  chatId: string,
  afterMessageId: string,
  limit = 50,
) {
  return requestAuth(
    `/chats/${chatId}/messages/after?messageId=${afterMessageId}&limit=${limit}`,
  );
}

export function getMessagesBefore(
  chatId: string,
  beforeMessageId: string,
  limit = 50,
) {
  return requestAuth(
    `/chats/${chatId}/messages/before?messageId=${beforeMessageId}&limit=${limit}`,
  );
}
