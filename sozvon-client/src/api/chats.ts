// sozvon-client/src/api/chats.ts
import { requestAuth } from "./http";

const API_URL = "http://92.127.177.190:8080";

export function createChat(toUserId: number) {
  return requestAuth("/chats/create", {
    method: "POST",
    body: JSON.stringify({ to_id: toUserId }),
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

export function sendMessage(
  chatId: string,
  text: string,
  replyToId?: string,
  forwardedFromId?: string,
) {
  return requestAuth(`/chats/${chatId}/messages`, {
    method: "POST",
    body: JSON.stringify({ text, replyToId, forwardedFromId }),
  });
}

// Отправка с файлами через multipart/form-data
export function uploadFiles(
  chatId: string,
  text: string,
  files: File[],
  replyToId?: string,
  forwardedFromId?: string,
  onProgress?: (percent: number) => void,
): Promise<any> {
  return new Promise((resolve, reject) => {
    const token = localStorage.getItem("token");
    const formData = new FormData();

    if (text.trim()) {
      formData.append("text", text.trim());
    }

    if (replyToId) formData.append("replyToId", replyToId);
    if (forwardedFromId) formData.append("forwardedFromId", forwardedFromId);

    files.forEach((file) => formData.append("files", file));

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
