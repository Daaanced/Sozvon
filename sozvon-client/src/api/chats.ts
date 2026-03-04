// sozvon-client/src/api/chats.ts
import { requestAuth } from './http'

// Создать чат (from_id и to_id — серверу нужны ID, но мы их берём из токена на сервере)
// from_id подставляет сервер из JWT, to_id берём из профиля найденного пользователя
export function createChat(toUserId: number) {
  return requestAuth('/chats/create', {
    method: 'POST',
    body: JSON.stringify({ to_id: toUserId })
  })
}

export function getChats() {
  return requestAuth('/chats')
}

export function getMessages(chatId: string, limit = 50, offset = 0) {
  return requestAuth(`/chats/${chatId}/messages?limit=${limit}&offset=${offset}`)
}

export function sendMessage(chatId: string, text: string) {
  return requestAuth(`/chats/${chatId}/messages`, {
    method: 'POST',
    body: JSON.stringify({ text })
  })
}