# Voice Service

Микросервис Sozvon, отвечающий за голосовые коммуникации в реальном времени. Он реализует голосовые комнаты, подключение пользователей к ним, WebRTC-сигнализацию и маршрутизацию аудиопотоков через собственный SFU (Selective Forwarding Unit).

В отличие от Chat Service, где WebSocket используется в основном для доставки событий, здесь WebSocket используется как сигнальный канал для установления и управления WebRTC-соединением, а сами аудиоданные передаются отдельным UDP-медиапотоком. Это прямо отражено в жизненном цикле Voice Service: после обмена offer/answer/ICE медиапоток идет отдельно от WebSocket.

# Основная функциональность
## Голосовые комнаты

Сервис позволяет создавать, получать список, просматривать и удалять голосовые комнаты:

	GET    /api/voice/rooms
	POST   /api/voice/rooms
	GET    /api/voice/rooms/{roomID}
	DELETE /api/voice/rooms/{roomID}

Комната содержит:

	id
	name
	created_by
	created_at
	peer_count
	peers

При создании комнаты сервис проверяет JWT пользователя и записывает его как создателя комнаты.

Количество одновременно существующих комнат и максимальное число участников ограничиваются конфигурацией:

	MaxRooms
	MaxPeersInRoom
	WebSocket-сигнализация

### Основной real-time интерфейс Voice Service — WebSocket:

`/ws`

Gateway проксирует внешний маршрут:

`/ws/voice → Voice Service /ws`

При подключении сервис получает JWT из query parameter:

`/ws?token=<JWT>`

После успешной проверки токена создается отдельная Session.

У сессии есть:

	peerID
	userID
	username
	WebSocket connection
	Peer
	context

peerID генерируется как UUID и является идентификатором конкретного подключения, а userID берется из JWT.

## Самостоятельная проверка JWT

Voice Service не обращается к Auth Service для каждой проверки токена.

Он использует тот же JWT secret, что и Gateway/Auth Service, и самостоятельно проверяет подпись и срок действия токена.

Это сделано специально, чтобы не добавлять дополнительный сетевой запрос и задержку на критическом пути установления голосового соединения.

Из JWT сервис получает:

	user_id
	login
	name

и использует эти данные для идентификации участника голосовой комнаты.

# WebRTC

Главная технологическая особенность сервиса — использование WebRTC через библиотеку Pion WebRTC.

После подключения клиента происходит примерно такая последовательность:

	Client
	│
	│ WebSocket
	▼
	Voice Service
	│
	├── JWT validation
	├── Join Room
	├── SDP Offer
	├── SDP Answer
	└── ICE Candidates
			│
			▼
		WebRTC connection
			│
			▼
		UDP audio stream

WebSocket в данном случае используется для signaling, а не для передачи самого голоса.

## SFU — Selective Forwarding Unit

В основе Voice Service находится собственная реализация SFU.

Главные компоненты:

	Engine
	│
	├── Room
	│     └── Peer
	│           └── PeerConnection
	│
	└── peerRoom

Engine управляет всеми комнатами, а каждая Room содержит участников Peer.

Каждый Peer представляет отдельного участника и содержит:

WebRTC PeerConnection;
входящий аудиотрек пользователя;
локальные треки для других участников;
канал сигнальных сообщений;
состояние mute/deafened;
буферизованные ICE/SDP данные.
Как работает передача голоса

Здесь реализована схема SFU:

              User A
                │
                │ audio
                ▼
          ┌────────────┐
          │    SFU     │
          └─────┬──────┘
                │
          ┌─────┴─────┐
          ▼           ▼
        User B      User C

Пользователь отправляет свой аудиопоток на Voice Service, а SFU пересылает его остальным участникам.

То есть участникам не приходится устанавливать отдельное прямое соединение с каждым другим участником.

При появлении нового аудиотрека Room автоматически распределяет его существующим участникам.

## RTP fan-out

Для эффективного распространения одного входящего RTP-потока используется trackFanout.

Логика:

	RemoteTrack
		│
		▼
	trackFanout
	 ┌──┼──┐
	 ▼  ▼  ▼
	 T1 T2 T3

Одна goroutine читает RTP-пакеты с TrackRemote, после чего каждый пакет записывается во все TrackLocalStaticRTP, соответствующие подписчикам.

Таким образом, Voice Service не декодирует и не кодирует аудио заново на каждом участнике, а выполняет RTP forwarding.

Это и есть ключевая идея SFU.

# Управление подписками

Когда новый пользователь подключается к комнате, сервис не только сообщает о нем существующим участникам, но и подписывает нового участника на уже существующие аудиотреки.

Когда участник покидает комнату:

	Peer
	↓
	RemovePeer
	↓
	unsubscribe от его track
	↓
	peer_left
	↓
	закрытие PeerConnection

Оставшимся пользователям отправляется событие:

	peer_left

## SDP negotiation

Для установки WebRTC-соединения используется стандартный обмен:

	Client → offer
	Server → answer

Когда на сервере появляется новый исходящий трек, требуется повторная negotiation:

	Server → re-offer
	Client → answer

Для этого реализован отдельный renegotiation worker.

Он следит за состоянием signaling state и выполняет повторную negotiation только в состоянии stable.

Это позволяет динамически добавлять и удалять аудиотреки участников без пересоздания всего WebRTC-соединения.

## ICE

Для установления сетевого соединения используется ICE.

Сервис поддерживает:

	STUN;
	TURN;
	внешний PublicIP;
	ограниченный диапазон UDP-портов.

ICE-кандидаты передаются через WebSocket.

Причем если ICE-кандидат приходит раньше RemoteDescription, он временно сохраняется в pendingICE, а затем применяется после установки SDP.

Это позволяет корректно обрабатывать порядок прихода сигнализационных сообщений.

## UDP-медиапорты

В конфигурации присутствует:

	UDPPortMin
	UDPPortMax

Эти порты предназначены именно для WebRTC медиапотоков.

Pion ограничивает диапазон ephemeral UDP ports через:

	SetEphemeralUDPPortRange(...)

Это особенно удобно для Docker/firewall, потому что серверу не требуется открывать произвольный диапазон UDP-портов.

Сетевая схема получается такой:

	Client
	│
	│ HTTPS / WSS
	▼
	Gateway
	│
	│ WS signaling
	▼
	Voice Service
	│
	│ WebRTC / UDP
	▼
	медиапоток

То есть голосовые данные не проходят через HTTP и не передаются внутри WebSocket. WebSocket нужен для управления WebRTC-соединением, а RTP/UDP — для самого медиа.

## STUN / TURN

Voice Service поддерживает настройку:

	STUNServers
	TURNServer
	TURNUser
	TURNPass

STUN используется для получения информации о доступных сетевых маршрутах и публичном адресе клиента.

TURN предусмотрен как relay-механизм для случаев, когда прямое установление WebRTC-соединения невозможно из-за NAT/firewall.

Также поддерживается настройка PublicIP сервера, которая используется при формировании ICE host candidates.

## Аудио

При подключении peer сервер добавляет audio transceiver:

	RTPCodecTypeAudio
	Direction: Recvonly

То есть сервер принимает аудио от клиента.

Затем для остальных участников создаются TrackLocalStaticRTP, через которые этот аудиотрек отправляется другим peer'ам.

Таким образом, текущая реализация в первую очередь ориентирована именно на голосовую связь.

### Mute и deafened

Сервис хранит состояние:

	Muted
	Deafened

для каждого peer.

При изменении состояния событие рассылается другим участникам комнаты:

	peer_muted
	peer_deafened

Это позволяет клиентам отображать:

	🎤 muted
	🔇 deafened

или аналогичные состояния в UI.

При этом состояние хранится внутри Peer, защищенного sync.RWMutex.

### Протокол сигнализации

Для общения клиента и Voice Service используется собственный JSON-протокол.

	Клиент → сервер

Поддерживаются:

	join
	leave
	offer
	answer
	ice_candidate
	mute
	deafened
	set_layer
	Сервер → клиент

Используются:

	room_state
	peer_joined
	peer_left
	offer
	answer
	ice_candidate
	peer_muted
	peer_deafened
	error

Сам набор типов сообщений явно определен в signal/types.go.

При подключении клиент сначала выполняет:

	join

после чего начинается WebRTC negotiation. VoiceWSHandler маршрутизирует входящие сообщения на соответствующие обработчики.

### Room State

После присоединения новый участник получает текущее состояние комнаты:

	room_state

в котором содержатся:

	room_id
	room_name
	peers[]

Каждый peer описывается:

	peer_id
	user_id
	username
	muted
	deafened

При этом собственный peer в room_state не включается.

Обработка отключений

Сервис отслеживает состояние WebRTC:

	PeerConnectionState
	ICEConnectionState
	SignalingState
	ICEGatheringState

При состояниях:

	failed
	disconnected
	closed

peer удаляется из комнаты.

Дополнительно существует отдельный pingLoop WebSocket-сессии для контроля доступности клиента.

При отключении:

	session closed
		↓
	LeaveRoom
		↓
	RemovePeer
		↓
	unsubscribe tracks
		↓
	notify peers

### Автоматическое удаление пустых комнат

Когда последний участник покидает комнату, она не удаляется мгновенно.

Запускается отложенная очистка:

30 секунд

Если за это время никто не присоединился обратно, комната удаляется из Engine.

Это позволяет избежать накопления пустых комнат и одновременно не удалять комнату мгновенно при кратковременном отключении.

### Simulcast

В архитектуре уже заложена поддержка simulcast:

	low
	medium
	high

Peer хранит предпочитаемый слой для каждого издателя:

	subscribedLayers

и поддерживает:

	set_layer

Но в текущей реализации это обозначено как задел на будущее, прежде всего для видео/демонстрации экрана, а текущий Voice Service работает с аудио.

# Конфигурация

Основные настройки Voice Service:

	Address
	JWTSecret

	STUNServers
	TURNServer
	TURNUser
	TURNPass

	PublicIP

	UDPPortMin
	UDPPortMax

	MaxRooms
	MaxPeersInRoom

То есть конфигурация позволяет отдельно задавать:

	сетевой адрес HTTP/WebSocket;
	секрет JWT;
	инфраструктуру STUN/TURN;
	публичный IP;
	диапазон UDP;
	ограничения голосовой системы.
	Надежность

Как и другие сервисы Sozvon, Voice Service использует:

	Logging middleware;
	Recovery middleware;
	HTTP timeouts;
	context;
	goroutines;
	mutex/sync.Map для конкурентного доступа;
	graceful shutdown.

При завершении сервиса сначала закрывается SFU Engine, который закрывает все существующие комнаты и их PeerConnection, а затем корректно останавливается HTTP-сервер.

Архитектура Voice Service

В упрощенном виде:

    	  ┌──────────────────┐
          │      Client      │
          └────────┬─────────┘
                   │
             HTTPS / WebSocket
                   │
                   ▼
          ┌──────────────────┐
      	  │     Gateway      │
          └────────┬─────────┘
                   │
                /ws/voice
                   │
                   ▼
        ┌─────────────────────┐
        │      Voice Service  │
        │                     │
        │  JWT + signaling    │
        │         │           │
        │         ▼           │
        │       Engine        │
        │         │           │
        │      ┌──┴──┐        │
        │      ▼     ▼        │
        │    Room  Room ...   │
        │     │               │
        │    Peer             │
        │     │               │
        │ PeerConnection      │
        │     │               │
        │ RTP/SFU fan-out     │
        └──────────┬──────────┘
                   │
                UDP/RTP
                   │
        ┌──────────┴──────────┐
        ▼                     ▼
    Client B              Client C

# Ключевая идея сервиса

Главное отличие Voice Service от остальных микросервисов Sozvon состоит в том, что он работает не просто как REST-сервис, а как stateful real-time WebRTC-сервис.

Его задача состоит из трех уровней:

1. Control Plane
   REST API → создание и управление комнатами

2. Signaling Plane
   WebSocket → join / SDP / ICE / mute / events

3. Media Plane
   WebRTC + RTP/UDP → передача голоса через SFU

Именно такое разделение наиболее точно описывает архитектуру представленного кода. В текущем варианте данные голосового трафика проходят через SFU Voice Service, а WebSocket используется для сигнализации и управления соединением.

# Основные технологии

	Go — серверная реализация;
	Pion WebRTC — WebRTC stack;
	SFU — маршрутизация RTP-потоков между участниками;
	WebRTC / RTP / SRTP — передача аудио;
	UDP — транспорт медиапотоков;
	ICE — установление соединения;
	STUN / TURN — работа с NAT и сложными сетями;
	WebSocket — signaling;
	JWT — аутентификация;
	Gorilla Mux — REST-маршрутизация;
	goroutines / channels / sync.Map / mutex — конкурентная обработка комнат, peer'ов и соединений.

