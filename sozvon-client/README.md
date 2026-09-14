# Sozvon Client — техническая документация

Клиентская часть мессенджера **Sozvon** — одностраничное веб-приложение на **React + TypeScript**. Предоставляет интерфейс для регистрации/авторизации, поиска пользователей, создания и просмотра чатов, обмена сообщениями, работы с файлами и голосовыми комнатами.

Клиент работает поверх единого backend **Gateway** и использует три механизма взаимодействия с сервером:

| Механизм | Назначение |
|---|---|
| **HTTP / REST** | получение и изменение данных |
| **WebSocket** | real-time события и голосовая сигнализация |
| **WebRTC** | передача аудио |

---

## 1. Архитектура клиента (обзор)

```
                          Sozvon Client
                               │
             ┌─────────────────┴───────────────────┐
             │                                     │
          React UI                            API layer
             │                                     │
    ┌────────┼──────────┐                 ┌────────┴─────────┐
    │        │          │                 │                  │
  Pages  Components  Context             HTTP            WebSocket
    │        │          │                 │                  │
    │        │          │                 ▼                  ▼
    │        │          │              Gateway             Gateway
    │        │          │                                    │
    │        │          │                                    ├─ Chat
    │        │          │                                    └─ Voice
    │        │          │
    └────────┴──────────┘
             │
        Browser API
             │
        WebRTC / Audio
```

**Итоговая архитектура (полная схема):**

```
                           SOZVON CLIENT
                                 │
                           React + TypeScript
                                 │
              ┌──────────────────┼──────────────────┐
              │                  │                  │
              ▼                  ▼                  ▼
          REST API           WebSocket            WebRTC
              │                  │                  │
              ▼                  ▼                  ▼
           Gateway            Gateway          Voice Service
              │                  │                  │
       ┌──────┼─────┐       ┌────┴──────┐           │
       │      │     │       │           │           │
      Auth   User  Chat    Chat       Voice       UDP/RTP
       │      │     │     Service    Service        │
       ▼      ▼     ▼       │           │           ▼
      auth profile  messages        signaling      audio
```

**Главная идея:** React отвечает за интерфейс и состояние, REST — за CRUD-операции, единый WebSocket — за real-time события, WebRTC — за передачу голосового медиапотока.

---

## 2. Маршрутизация (React Router)

SPA, точка входа `src/main.tsx` — создает React root и подключает `RouterProvider`.

| Маршрут | Назначение |
|---|---|
| `/` | redirect на `/login` |
| `/login` | авторизация |
| `/register` | регистрация |
| `/app` | основное приложение |
| `/app/chats/:chatId` | конкретный чат |
| `/app/rooms` | голосовые комнаты |

Защищенные страницы находятся внутри `ProtectedRoute`: если JWT отсутствует в `localStorage`, пользователь перенаправляется на `/login`.

---

## 3. Авторизация и JWT на клиенте

### 3.1 Вход

```
	Login
	↓
	POST /api/auth/login
	↓
	Gateway
	↓
	Auth Service
	↓
	JWT
	↓
	localStorage
```

Токен сохраняется: `localStorage.setItem("token", res.token)`, после чего происходит переход `/login → /app`. `ProtectedRoute` проверяет наличие токена перед отображением защищенной части приложения.

### 3.2 Использование JWT

`ChatContext` разбирает payload JWT и получает `user_id`, `login` — это дает информацию о текущем пользователе без отдельного запроса.

Для авторизованных HTTP-запросов функция `requestAuth()` автоматически добавляет заголовок `Authorization: Bearer <token>`.

---

## 4. HTTP API layer

Разделение по функциональности:

```
	src/api/
	├── http.ts
	├── auth.ts
	├── users.ts
	├── chats.ts
	└── voice.ts
```

Это отделяет UI-компоненты от сетевого взаимодействия:

```
Component → createChat() → requestAuth() → fetch() → Gateway
```

### 4.1 HTTP-клиент (`http.ts`)

Базовый адрес задается через `VITE_API_URL`. Реализованы три типа запросов:

| Функция | Назначение |
|---|---|
| `request()` | обычные запросы без JWT |
| `requestAuth()` | автоматически добавляет JWT |
| `requestForm()` | преимущественно для `multipart/form-data` (загрузка файлов) |

---

## 5. Чаты — компонент `Chat`

`Chat` — контейнер конкретного чата: связывает загрузку истории, отправку и изменение сообщений, работу с файлами, ответы, пересылку, удаление, unread-состояние и управление прокруткой.

### 5.1 Композиция

```
Chat
 │
 ├── useChatMessages   → история + pagination + scroll
 │
 ├── useChatActions    → send / edit / delete / reply
 │
 ├── useFileUpload     → подготовка и загрузка файлов
 │
 ├── useVisibleMessages → определение прочитанных сообщений
 │
 ├── ChatTopBar
 │
 ├── ChatMessages
 │      └── MessageBubble
 │
 └── ChatInput
```

`Chat` получает `chatId` из маршрута `/app/chats/:chatId` и находит соответствующий чат в `ChatContext`. Для direct-чата определяется собеседник, для group — используются название и групповой аватар. В верхней панели: аватар, имя/название группы, поиск, вызов, настройки.

### 5.2 История сообщений — `useChatMessages`

Отвечает за: первоначальную загрузку, подгрузку старых/новых сообщений, загрузку контекста вокруг сообщения, переход к сообщению/к последнему сообщению, состояние загрузки, управление `scrollIntent`. Компонент `Chat` не работает напрямую с HTTP API истории — получает готовое состояние и callback'и.

#### Двунаправленная пагинация

```
        старые
          ↑
     loadMore()
          ↑
─────────────────────
   текущая история
─────────────────────
          ↓
   loadMoreBottom()
          ↓
        новые
```

`ChatMessages` использует два `IntersectionObserver` (top sentinel, bottom sentinel): попадание верхнего в viewport запрашивает предыдущую историю, нижнего — более новую. Особенно полезно после перехода к непрочитанным или восстановления позиции.

#### Сохранение позиции скролла

При подгрузке старых сообщений в начало списка сохраняется предыдущий `scrollHeight`. После загрузки:

```
новый scrollTop = новый scrollHeight - старый scrollHeight
```

Используется `useLayoutEffect`, чтобы корректировка происходила до отображения — без заметного скачка.

#### Автопереход к непрочитанным

`scrollIntent` поддерживает типы: `bottom`, `unread`, `message`.

```
unread → найти firstUnread → scrollIntoView() → показать "Новые сообщения"
```

`ChatMessages` хранит `firstUnreadIdRef`, над первым непрочитанным отрисовывается разделитель `── Новые сообщения ──`. После перехода дополнительно проверяется нижний sentinel для подгрузки продолжения истории.

#### Отслеживание прочитанных — `useVisibleMessages`

Для каждого сообщения передаются `id`, `createdAt`, DOM element; при попадании в область видимости вызывается `handleMarkRead(chatId, id)`.

```
message visible → markRead() → POST /api/chats/{chatId}/read → last_read_message_id
```

Состояние прочтения определяется фактической видимостью, а не моментом открытия чата.

### 5.3 Отправка и ввод текста

`ChatInput` предоставляет: `text`, `pendingFiles`, `replyTo`.

```
Send → handleSend() → useChatActions.send()
```

После отправки очищаются текст и подготовленные файлы. Если была доступна нижняя часть истории — используется `setTimeout`, чтобы дать WebSocket время доставить сообщение, затем выполняется переход вниз.

**Правила ввода:**
- Enter → отправка, Shift+Enter → новая строка
- лимит текста — 4000 символов
- автофокус поля при выборе ответа

### 5.4 Ответы на сообщения

```
↩ Ответить → сообщение сохраняется в replyTo → ChatInput показывает "↩ Ответ: текст..."
```

Пользователь может отменить режим ответа. `replyToId` передается на backend. Сообщения поддерживают `replyToId`/`replyToMessage`, `MessageBubble` показывает preview исходного; клик по preview вызывает `onScrollToMessage()`.

### 5.5 Редактирование

`✏️ Изменить` → `EditModal` (textarea, Сохранить/Отмена) → `editMessage()` → рассылка изменения через WebSocket. Отредактированное сообщение помечается `(изменено)`.

### 5.6 Удаление

`🗑 Удалить` → после удаления backend уведомляет через WebSocket, клиент отображает «Сообщение удалено» вместо содержимого; вложения удаленного сообщения не выводятся.

### 5.7 Пересылка

`ForwardModal`: выбор чата, добавление комментария, отправка через `forwardMessages()`. После успеха вызывается `notifyOwnMessage()`, чтобы целевой чат переместился наверх локального списка.

### 5.8 Вложения

`ChatInput` поддерживает до `MAX_FILES = 4`. Допустимые типы: images, videos, PDF, DOC/DOCX, XLS/XLSX, ZIP, RAR, TXT.

Перед отправкой формируется `PendingFile`: `File`, `previewUrl`, `width`, `height`. Для изображений preview создается через `URL.createObjectURL()`.

**Drag & Drop:** на `window` устанавливаются `dragover`/`drop`; файлы после drop проходят ту же подготовку, что и из file picker.

**Вставка из Clipboard:** `ChatInput` обрабатывает `paste`:
```
clipboard → File → PendingFile → preview
```
Дополнительно поддерживается GIF из HTML clipboard.

**Превью:** для изображений и видео вычисляется `aspectRatio = width / height` для пропорционального отображения.

**Upload progress:** используется `XMLHttpRequest` (не `fetch`) ради `xhr.upload.onprogress` → отображение progress bar (`0% → 100%`).

### 5.9 Отображение вложений

`MessageBubble` делит attachments на три категории: `image`, `video`, `other files`.

- **Изображения:** одно — большое preview; несколько — сетка 2×2.
- **Видео:** `<video controls>`.
- **Файлы:** ссылка вида `📎 filename.pdf 4.2 MB`, открытие в новой вкладке.
- **Lightbox:** клик по изображению → `onImageClick(url)` → `lightboxUrl` → `Lightbox` (просмотр поверх интерфейса, закрытие кликом/кнопкой).

### 5.10 Группировка сообщений и время

`isGroupStart` определяет начало новой группы: сменился отправитель или изменилась дата.

```
Alex    Привет
        Как дела?
        Что сегодня делаешь?

Bob     Всё хорошо
        А у тебя?
```

Первое сообщение группы показывает аватар/имя/полную дату, остальные — компактное время.

**Форматирование времени:** `formatTime()` → `14:37`; `formatFullDate()` → `10.09.2026, 14:37`.

### 5.11 Меню сообщения

При наведении — overlay `↩ ↪ •••` с действиями: Ответить, Переслать, Изменить, Удалить (последние два — только для своих сообщений). Dropdown рендерится через `createPortal` в `document.body` — избегает проблем с overflow/clipping/позиционированием.

### 5.12 Удаленные пользователи

`getSafeUser(senderId)` — сообщение остается отображаемым, даже если профиль автора недоступен; используется placeholder `DELETED_USER`.

### 5.13 Верхняя панель — `ChatTopBar`

Отвечает за название чата, аватар, поиск сообщений, звонок, настройки. Для direct — имя пользователя; для group — `groupInfo.name`/`groupInfo.picture`.

> Поиск сообщений — пока только UI-элемент ввода, логика поиска не реализована. Кнопка Call вызывает переданный callback, но сам запуск голосового вызова из компонента не выполняется.

### 5.14 Модель данных

```
Message
├── id
├── chatId
├── senderId
├── text
├── replyToId
├── replyToMessage
├── forwardedFrom
├── editedAt
├── deletedAt
├── attachments
└── createdAt
```

```
Attachment
├── id
├── fileName
├── mimeType
├── size
├── url
├── width
└── height
```

Единая модель поддерживает обычные сообщения, ответы, forwarding, редактирование, удаление и медиа.

### 5.15 API чатов (`src/api/chats.ts`)

```
createChat()
createGroupChat()
getChats()

getMessages()
getMessagesBefore()
getMessagesAfter()
getMessagesContext()

sendMessage()
uploadFiles()

forwardMessages()
editMessage()
deleteMessage()

markRead()
getChatInfo()
getUnreadMessages()
```

### 5.16 Создание чатов

**Direct:**
```json
{ "to_id": 123 }
```

**Group:**
```json
{ "name": "Developers", "member_ids": [12, 15, 21] }
```

`CreateGroupModal` позволяет выбирать пользователей; ограничение на клиенте — **максимум 9 выбранных участников**.

### 5.17 Цепочка отправки сообщения (полная схема)

```
                 React UI
                    │
                ChatInput
                    │
                    ▼
              useChatActions
                    │
                    ▼
              API / upload
                    │
                    ▼
                 Gateway
                    │
                    ▼
              Chat Service
                    │
             ┌──────┴──────┐
             │             │
          Database       WebSocket
             │             │
             │             ▼
             │          Gateway
             │             │
             └─────────────┤
                           │
                           ▼
                        Client
```

**Загрузка истории:**
```
Chat → useChatMessages → GET /messages → Gateway → Chat Service → response → messages state → ChatMessages → MessageBubble
```

---

## 6. ChatContext — глобальное состояние чатов

Хранит: `chats`, `users`, `myId`, `myLogin`, `me`, `unread`.

Операции: `loadUser`, `handleMarkRead`, `notifyOwnMessage`, `getSafeUser`, `setActiveChat`.

Позволяет разным компонентам использовать общую модель состояния вместо независимого хранения данных.

### 6.1 Кэширование пользователей

Локальный кэш `Record<number, User>`; пользователь загружается по `user_id` только при необходимости. `loadingUsersRef` предотвращает одновременную отправку нескольких одинаковых запросов — снижает нагрузку на User Service.

### 6.2 Удаленные пользователи

Placeholder `DELETED_USER` — интерфейс отображает «Deleted» с отдельным изображением; полезно для старых сообщений/чатов с уже удаленным пользователем.

### 6.3 Состояние unread

Отслеживает `lastMessageId`, `lastReadMessageId`, `unread`. При получении `message:new` клиент обновляет последнее сообщение и помечает чат непрочитанным, если он сейчас не открыт (если пользователь внутри чата — индикатор не показывается).

Sidebar в итоге отображает:
```
Chat A
Chat B ●
Chat C
```

---

## 7. WebSocket (`src/services/ws.ts`)

Создается **единственное** WebSocket-соединение к `Gateway /ws`. API:

```
connectWS()
disconnectWS()
sendWS()
onWSMessage()
```

Это глобальный транспорт, которым пользуются и `ChatContext`, и `VoiceClient`.

### 7.1 Единое соединение

После входа в `/app`, `AppShell` вызывает `connectWS(token)` — WebSocket создается на уровне приложения, а не отдельно каждым компонентом.

```
                   Browser
                      │
                      │ WebSocket
                      ▼
                   Gateway
                      │
             ┌────────┴────────┐
             ▼                 ▼
       Chat Service       Voice Service
```

### 7.2 Real-time события чатов

`ChatContext` подписывается через `onWSMessage()` на: `chat:created`, `chat:activated`, `message:new`.

Пример для `message:new`:
```
WebSocket → ChatContext → обновляет chat.updatedAt → обновляет lastMessageId → перемещает чат вверх → ставит unread
```

Sidebar обновляется без повторной загрузки всего списка.

---

## 8. Загрузка файлов (детали)

Отдельный `XMLHttpRequest` вместо `fetch` — ради `xhr.upload.onprogress`.

`uploadFiles()` передает: `text`, `replyToId`, `files[]`, `meta`. UI отображает:
```
Uploading...
████████████░░░░ 75%
```

---

## 9. Профиль пользователя (`src/api/users.ts`)

```
searchUser()
getUserById()
updateUser()
uploadAvatar()
deleteAvatar()
deleteUser()
```

Модель профиля:
```
id
login
name
email
info
picture
created_at
updated_at
```

### 9.1 Аватары

Отображаются: в списке чатов, в профиле, в информации о пользователе, при создании группового чата. При отсутствии изображения — fallback с первой буквой имени или placeholder. Загрузка — через `FormData`.

### 9.2 Settings

`SettingsModal` объединяет разделы: Profile, Appearance, Sound. Через него выполняется logout:

```
localStorage.removeItem("token")
disconnectWS()
navigate("/login")
```

---

## 10. UI Layout

Основная структура — `AppShell`, три области:

```
┌────────────┬────────────────────────────┬──────────────┐
│            │                            │              │
│  Sidebar   │          Content           │  User Info   │
│            │                            │              │
│  Chats     │        Chat / Main         │  Profile     │
│  Voice     │                            │  Members     │
│            │                            │              │
└────────────┴────────────────────────────┴──────────────┘
```

Правая панель UserInfo включается только от определенной ширины окна: `USERINFO_BREAKPOINT = 1100`.

### 10.1 Изменяемая ширина Sidebar

```
SIDEBAR_MIN     = 180
SIDEBAR_DEFAULT = 260
SIDEBAR_MAX     = 400
```

Реализовано через `mousedown`/`mousemove`/`mouseup` и React state.

### 10.2 Sidebar

Объединяет: Direct Messages, Voice rooms, Search users, Chat list, Current user, Voice controls, Settings. Отображает текущую голосовую комнату, например:
```
Connected ●
My Voice Room

🖥️   📵
```

---

## 11. Voice Client (`src/api/voice.ts`)

Голосовая часть построена отдельно от Chat API. Основной класс — `VoiceClient`, отвечает за: подключение к voice room, получение микрофона, создание `RTCPeerConnection`, SDP negotiation, ICE, прием remote audio, mute/deafen, обработку событий участников.

### 11.1 Жизненный цикл подключения

```
joinRoom()
   ↓
WebSocket join
   ↓
initPC()
   ↓
getUserMedia()
   ↓
addTrack(audio)
   ↓
createOffer()
   ↓
send offer
   ↓
Voice Service
   ↓
answer
   ↓
setRemoteDescription()
   ↓
ICE exchange
   ↓
WebRTC connected
   ↓
audio
```

### 11.2 WebRTC API браузера

```
RTCPeerConnection
MediaStream
MediaStreamTrack
navigator.mediaDevices.getUserMedia()
```

Захват только микрофона:
```js
getUserMedia({ audio: true, video: false });
```

### 11.3 Negotiation и glare

Реализованы: `createOffer()`, `createAnswer()`, `setLocalDescription()`, `setRemoteDescription()`.

Обрабатывается ситуация **glare** (обе стороны одновременно начинают negotiation): при получении server offer в состоянии `have-local-offer` клиент выполняет `rollback`, затем принимает удаленный offer — это синхронизирует negotiation при динамическом добавлении треков сервером.

### 11.4 ICE

Клиент получает кандидатов через `pc.onicecandidate` и передает их по общему WebSocket:
```json
{
  "type": "ice_candidate",
  "payload": { "candidate": "...", "sdp_mid": "...", "sdp_mline_index": 0 }
}
```

Обратные кандидаты от сервера добавляются через `pc.addIceCandidate()`.

```
Signaling → WebSocket → SDP + ICE
Media     → WebRTC
```

### 11.5 Прием голоса от других участников

Обрабатывается `pc.ontrack`: при появлении удаленного трека создается `HTMLAudioElement`, `srcObject` устанавливается в полученный `MediaStream`, воспроизведение автоматическое (`audio.autoplay = true`). Для каждого peer — отдельный элемент:

```
Peer B ── audio ──► Audio Element B
Peer C ── audio ──► Audio Element C
Peer D ── audio ──► Audio Element D
```

---

## 12. Типы голосовой части (`voice.types.ts`)

### `RoomInfo`
```ts
export interface RoomInfo {
  id: string;
  name: string;
  created_by: string;
  created_at: string;
  peer_count: number;
  peers: PeerInfo[];
}
```

### `PeerInfo`
```ts
export interface PeerInfo {
  peer_id: string;
  user_id: string;
  username: string;
  muted: boolean;
  deafened: boolean;
}
```

Два разных идентификатора: `peer_id` — WebRTC-peer внутри комнаты, `user_id` — ID пользователя Sozvon.

### `VoiceEvents`
```ts
export interface VoiceEvents {
  onRoomState?: (peers: PeerInfo[]) => void;
  onPeerJoined?: (peer: PeerInfo) => void;
  onPeerLeft?: (peerId: string) => void;
  onPeerMuted?: (peerId: string, muted: boolean) => void;
  onPeerDeafened?: (peerId: string, deafened: boolean) => void;
  onTrack?: (peerId: string, stream: MediaStream) => void;
  onError?: (code: string, message: string) => void;
  onConnected?: () => void;
  onDisconnected?: () => void;
}
```

Особенно важен `onTrack` — при получении `MediaStream` от другого пользователя клиент подключает поток к `<audio>`.

---

## 13. VoiceRoomsPage.tsx

Страница управления голосовыми комнатами (UI-слой, не сама WebRTC-логика):

```
Получить комнаты → Показать комнаты → Создать комнату → Присоединиться → Показать участников → Выйти → Удалить комнату
```

### 13.1 Локальное состояние

```ts
const [rooms, setRooms] = useState<RoomInfo[]>([]);
const [loading, setLoading] = useState(true);
const [error, setError] = useState("");
const [newRoomName, setNewRoomName] = useState("");
const [creating, setCreating] = useState(false);
const [showCreateInput, setShowCreateInput] = useState(false);
```

### 13.2 Контекст

`const { me } = useChatContext();` — текущий пользователь, используется для добавления себя в список участников активной комнаты.

```ts
const {
  activeRoomId, connecting, muted, deafened,
  leaveRoom, joinRoom, peers,
} = useVoiceContext();
```

| Поле | Значение |
|---|---|
| `activeRoomId` | ID комнаты, в которой сейчас пользователь |
| `connecting` | идет ли подключение |
| `muted` | выключен ли микрофон |
| `deafened` | выключен ли звук у пользователя |
| `joinRoom` / `leaveRoom` | подключение / выход |
| `peers` | участники текущего голосового соединения |

### 13.3 Загрузка и автообновление комнат

```
VoiceRoomsPage → getRooms() → Voice API → Voice Service → rooms → setRooms() → React render
```

```ts
useEffect(() => {
  fetchRooms();
  const interval = setInterval(fetchRooms, 300000); // 5 минут
  return () => clearInterval(interval);
}, [fetchRooms]);
```

### 13.4 Создание и удаление комнаты

**Создание:**
```
Введите название → createRoom(name) → Voice Service создает Room → fetchRooms() → обновляется список
```

**Удаление** (`handleDeleteRoom`): если пользователь находится в удаляемой комнате — сначала `leaveRoom()`, затем `deleteRoom(roomId)`, затем `fetchRooms()`.

### 13.5 Формирование списка участников

```ts
const livePeers = activeRoomId
  ? [
      { peer_id: String(me!.id), user_id: String(me!.id), username: me!.name, muted, deafened },
      ...peers,
    ]
  : peers;
```

`peers` из `VoiceContext` содержит только других участников — текущий пользователь добавляется вручную:

```
livePeers
├── Ivan ← текущий пользователь
├── Alex
├── Bob
└── John
```

### 13.6 Отрисовка комнаты и кнопка Join/Leave

`isActive = room.id === activeRoomId` — активная комната получает стиль `s.roomCardActive`.

```tsx
{isActive ? (
  <button onClick={leaveRoom}>Leave</button>
) : (
  <button onClick={() => joinRoom(room.id, room.name)}>Join</button>
)}
```

Сама WebRTC-сессия создается не здесь, а внутри `VoiceContext`/`VoiceClient`.

### 13.7 PeerBadge

Отдельный компонент для одного участника. Из имени берется первая буква (`Alexander → A`):

```
┌──────────────┐
│ A  Alexander │
└──────────────┘
```

**Индикаторы состояния:**
```tsx
{peer.deafened && <span title="Deafened">🧱</span>}
{peer.muted && <span title="Muted">🔇</span>}
{live && !peer.muted && <span title="Speaking">🎙️</span>}
```

> Важный нюанс: условие `live && !peer.muted` означает «не muted», а не факт реальной речи — это не полноценный voice activity detection.

### 13.8 Роль страницы в архитектуре

```
VoiceRoomsPage
   │
   ├── REST
   │    ├── getRooms()
   │    ├── createRoom()
   │    └── deleteRoom()
   │
   └── VoiceContext
        ├── joinRoom()
        ├── leaveRoom()
        ├── muted
        ├── deafened
        └── peers
             │
             ▼
         VoiceClient
             │
      ┌──────┴──────┐
      │             │
  WebSocket       WebRTC
  signaling        media
      │             │
   Gateway     Voice Service
```

`VoiceRoomsPage` не занимается WebRTC напрямую — отвечает за отображение комнат, создание/удаление, присоединение/выход, отображение участников и состояния mute/deafen/подключения. Низкоуровневая голосовая логика вынесена отдельно.

---

## 14. VoiceContext

Хранит: `activeRoomId`, `activeRoomName`, `peers`, `muted`, `deafened`, `callActive`, `connecting`.

Предоставляет: `joinRoom()`, `leaveRoom()`, `toggleMute()`, `toggleDeafen()`.

Низкоуровневая WebRTC-логика отделена от React UI.

---

## 15. Mute / Deafen (клиент)

| Действие | Эффект |
|---|---|
| **Mute** | `MediaStreamTrack.enabled = false` + сигнал `mute` — прекращает отправку собственного микрофона |
| **Deafen** | все аудиоэлементы получают `audio.muted = true` — прекращает воспроизведение входящего звука |

В текущей реализации `deafen` может автоматически включать `mute`.

---

## 16. Голосовые комнаты — сводка

Страница `/app/rooms`, API:
```
getRooms()
createRoom()
getRoom()
deleteRoom()
```

Отделяет управление комнатами через REST от голосовой связи через WebSocket/WebRTC.

---

## 17. Разделение REST / WebSocket / WebRTC

```
REST
 ├── login/register
 ├── users
 ├── chats
 ├── messages
 ├── files
 └── voice rooms

WebSocket
 ├── message:new
 ├── message:edited
 ├── chat:created
 ├── typing
 └── WebRTC signaling

WebRTC
 └── audio media
```

Это одна из главных архитектурных особенностей клиента.

---

## 18. Технологический стек

**Frontend**
- React, TypeScript
- React Router, Vite
- JSX/TSX
- React Context API
- React Hooks (`useState`, `useEffect`, `useRef`, `useCallback`)

**HTTP**
- Browser `fetch`
- `XMLHttpRequest` для upload progress
- REST API, JSON, `FormData`
- JWT Bearer authentication

**Real-time**
- WebSocket (единое соединение с Gateway)
- event-based архитектура, обработчики через подписки

**Voice**
- WebRTC, `RTCPeerConnection`, `MediaStream`, `getUserMedia`
- SDP, ICE, STUN

**Состояние**
- React Context, `useState`, `useRef`, `localStorage`

**Сборка**
- Vite, TypeScript, ES modules
