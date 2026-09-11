Gateway 

Центральный входной микросервис Sozvon, через который клиент взаимодействует с остальными backend-сервисами. Он выступает в роли API Gateway и WebSocket Gateway: принимает внешние HTTP/WebSocket-запросы, проверяет JWT, определяет целевой микросервис и проксирует запрос внутри Docker-сети.

Главная задача Gateway — скрыть внутреннюю структуру микросервисов от клиента. Клиенту не нужно знать адреса и порты Auth, User, Chat и Voice Service: он работает только с Gateway.

Упрощенно:

                        ┌─────────────────┐
                        │     Client      │
                        └────────┬────────┘
                                 │
                         HTTPS / WebSocket
                                 │
                                 ▼
                       ┌──────────────────┐
                       │     Gateway      │
                       └────────┬─────────┘
                                │
              ┌─────────────────┼─────────────────┐
              ▼                 ▼                 ▼
        Auth Service       User Service      Chat Service
                                                  │
                                                  ▼
                                           Voice Service

# Основная функциональность
Единая точка входа

Gateway предоставляет единый внешний API для всех сервисов.

Маршрутизация построена по URL:

	/api/auth/...   → Auth Service
	/api/users/...  → User Service
	/api/static/... → User Service
	/api/chats/...  → Chat Service
	/api/media/...  → Chat Service
	/api/voice/...  → Voice Service

Такая схема позволяет клиенту обращаться, например, к:

	/api/auth/login
	/api/users/...
	/api/chats/...
	/api/voice/rooms

не обращаясь напрямую к портам отдельных контейнеров.

## HTTP reverse proxy

Основная работа Gateway выполняется через HTTP-проксирование.

Для каждого запроса он:

+ получает HTTP-запрос клиента;
+ определяет целевой микросервис;
+ создает новый http.Request;
+ передает исходный HTTP method, URI и body;
+ копирует заголовки;
+ выполняет запрос к внутреннему сервису;
+ получает ответ;
+ копирует status code, headers и body обратно клиенту.

То есть:

	Client
	│
	│ GET /api/users/alex
	▼
	Gateway
	│
	│ GET /api/users/alex
	▼
	User Service
	│
	│ response
	▼
	Gateway
	│
	│ response
	▼
	Client

Это фактически собственный reverse proxy на Go, реализованный через net/http.

Для HTTP-клиента Gateway настроен connection pool:

	MaxIdleConns = 100
	MaxIdleConnsPerHost = 10
	IdleConnTimeout = 90s

Также каждый proxy request имеет timeout 30 секунд.

## WebSocket Gateway

Особенно интересна WebSocket-часть.

Gateway поддерживает:

`/ws`
`/ws/voice`

При этом /ws используется как единое WebSocket-соединение клиента, через которое Gateway может взаимодействовать как с Chat Service, так и с Voice Service.

Единое WebSocket-соединение для клиента

При подключении к:

`/ws?token=<JWT>`

# Gateway:

	получает JWT;
	валидирует его;
	устанавливает WebSocket-соединение с клиентом;
	автоматически подключается к Chat Service;
	создает клиентскую WebSocket-сессию;
	запускает отдельные goroutine для обмена данными.

Внутри Gateway один пользователь может иметь:

	client connection
		│
		├── chatConn
		│
		└── voiceConn

voiceConn создается лениво — только когда пользователь действительно начинает использовать голосовую часть.

Это позволяет не устанавливать дополнительное соединение с Voice Service для каждого обычного пользователя.

## Маршрутизация WebSocket-сообщений

Gateway анализирует содержимое WebSocket-сообщения и определяет, куда его отправить.

Для этого используется routeMessage().

Если сообщение содержит:

	{
	"service": "voice"
	}

оно направляется в Voice Service.

Кроме этого, Gateway распознает стандартные WebRTC signaling-сообщения:

	join
	leave
	offer
	answer
	ice_candidate
	mute
	deafened
	set_layer

и также направляет их в Voice Service. Остальные сообщения считаются сообщениями Chat Service.

Получается:

                Client
                   │
                   ▼
              Gateway /ws
                   │
          ┌────────┴────────┐
          │                 │
       Chat WS           Voice WS
          │                 │
          ▼                 ▼
     Chat Service      Voice Service

WebSocket proxy для Chat Service

После подключения Gateway устанавливает внутреннее WebSocket-соединение:

`/ws`

с Chat Service.

Сообщения:

`Client → Gateway → Chat Service`

а ответы:

`Chat Service → Gateway → Client`

Для этого работают отдельные goroutine.

readFromClient() получает сообщения от клиента и определяет целевой сервис, а readFromChatService() принимает события от Chat Service и передает их клиенту.

WebSocket proxy для Voice Service

Для голоса реализована аналогичная схема:

	Client
	│
	│ WebSocket signaling
	▼
	Gateway
	│
	│ WebSocket
	▼
	Voice Service

При необходимости Gateway устанавливает voiceConn и начинает передавать сообщения между клиентом и Voice Service.

При этом Gateway не передает сам голосовой UDP/RTP-трафик. Его задача — только проксировать WebSocket signaling.

Это важное архитектурное разделение:

	Gateway
	│
	└── WebSocket signaling
			│
			▼
		Voice Service
			│
			└── WebRTC / UDP / RTP

Отдельный /ws/voice

Помимо общего /ws, реализован отдельный endpoint:

`/ws/voice`

Он устанавливает соединение клиента напрямую через Gateway с Voice Service без Chat Service.

Таким образом, в текущей архитектуре существует два варианта работы с voice signaling:

	/ws
	└── Gateway маршрутизирует сообщения
		├── Chat Service
		└── Voice Service

	/ws/voice
	└── Gateway → Voice Service

Основной вариант /ws удобен как единая точка WebSocket-подключения клиента.

# JWT

Gateway самостоятельно валидирует JWT.

Он использует тот же секретный ключ, что и Auth Service, поэтому может проверить подпись токена без обращения к Auth Service. В Claims содержатся:

	Login
	UserID
	Name
	RegisteredClaims

При проверке контролируется:

корректность подписи;
допустимый алгоритм HMAC;
срок действия токена;
наличие обязательного login claim.

Таким образом:

	Client
	│
	│ JWT
	▼
	Gateway
	│
	└── ValidateJWT

а для внутренних WebSocket-соединений тот же токен передается дальше сервисам, которые также могут валидировать его самостоятельно.

## Почему Gateway валидирует JWT

Это дает Gateway возможность отклонить явно неавторизованный запрос еще до проксирования.

Однако в текущем коде Gateway в основном выступает как пограничная точка проверки, а не как единственный механизм безопасности.

Например, Chat и Voice Service также умеют самостоятельно проверять JWT.

Получается несколько уровней:

	Client
	↓
	Gateway
	└── JWT validation
			↓
		Service
			└── JWT validation

Это повышает независимость микросервисов и не заставляет доверять исключительно Gateway.

## CORS

Gateway выполняет CORS на внешнем уровне с помощью библиотеки:

`github.com/rs/cors`

Конфигурация задает:

	AllowedOrigins
	AllowCredentials
	AllowedMethods
	AllowedHeaders

Поэтому CORS не требуется отдельно реализовывать в каждом backend-сервисе.

Схема:

	Browser
	│
	▼
	Gateway
	│
	├── CORS
	├── JWT
	└── Routing
			│
			▼
		services

# Middleware

Gateway использует два собственных middleware.

## Logging

Для каждого запроса записываются:

	method
	URI
	HTTP protocol
	status
	duration
	IP

Это дает централизованный лог входящего трафика.

## Recovery

Middleware перехватывает panic, выводит stack trace в лог и возвращает:

500 Internal Server Error

В результате ошибка одного обработчика не должна привести к падению всего Gateway.

## Поддержка WebSocket через middleware

При использовании WebSocket недостаточно простого http.ResponseWriter, поскольку HTTP-соединение необходимо переключить в WebSocket.

Поэтому loggingResponseWriter дополнительно реализует:

`http.Hijacker`

что позволяет WebSocket-соединению корректно выполнить Upgrade.

Также реализован http.Flusher, необходимый для корректной поддержки потоковой передачи данных.

Это важная техническая деталь, поскольку Middleware оборачивает ResponseWriter, но при этом не должен ломать WebSocket upgrade.

## Контроль WebSocket-соединений

Для клиентского WebSocket используются:

writeWait = 10s
pongWait = 60s
pingPeriod = 54s
maxMessageSize = 512 KB

Gateway отправляет периодические Ping, а PongHandler обновляет deadline.

При потере соединения выполняется cleanup:

	Client
	↓
	cleanup()
	├── close client connection
	├── close chat connection
	└── close voice connection

## Буферизация

Между Gateway и клиентом используется канал:

send chan []byte

размером 256 сообщений.

Если канал полностью заполнен, Gateway не блокируется бесконечно, а может отбросить сообщение и записать предупреждение в лог.

То есть обработка WebSocket построена асинхронно и использует goroutines и channels.

## Работа в Docker / микросервисной сети

В конфигурации Gateway явно задаются URL сервисов:

AuthServiceURL
UserServiceURL
ChatServiceURL
VoiceServiceURL

В Docker Compose они могут указывать на имена контейнеров/сервисов, например:

auth-service:8082
user-service:8083
chat-service:8084
voice-service:8085

Поэтому внутренние порты сервисов можно не публиковать наружу.

## Внешняя схема:

	Internet
	│
	▼
	Gateway :8080
	│
	├── auth-service:8082
	├── user-service:8083
	├── chat-service:8084
	└── voice-service:8085

Это дает одно из главных преимуществ Gateway: внешний клиент видит только одну backend-точку.

# Health Check

Gateway имеет:

`/api/health`

который возвращает статус самого Gateway:

	{
	"status": "ok",
	"service": "gateway"
	}

Это может использоваться Docker, reverse proxy, мониторингом или системой деплоя для проверки работоспособности Gateway.

# Graceful shutdown

При получении:

	SIGINT
	SIGTERM

Gateway выполняет:

	получение сигнала
			↓
	context.WithTimeout(30s)
			↓
	srv.Shutdown()
			↓
	корректное завершение HTTP-сервера

Таким образом, сервис рассчитан на корректное завершение контейнера, а не на принудительное обрывание всех соединений.

# Итоговая архитектура Gateway

В контексте всего Sozvon его роль можно представить так:

                         ┌──────────────┐
                         │    Client    │
                         └──────┬───────┘
                                │
                         HTTPS / WSS
                                │
                                ▼
                    ┌──────────────────────┐
                    │       Gateway        │
                    │                      │
                    │  CORS                │
                    │  JWT validation      │
                    │  HTTP routing        │
                    │  WS routing          │
                    │  Logging             │
                    │  Recovery            │
                    └───────┬──────────────┘
                            │
          ┌─────────────────┼────────────────────┐
          │                 │                    │
          ▼                 ▼                    ▼
    Auth Service       User Service         Chat Service
                                                 │
                                                 │ WS
                                                 ▼
                                            Voice Service
                                                 │
                                                 │ WebRTC
                                                 ▼
                                              UDP/RTP
# Главное назначение

Gateway выполняет четыре основные функции:

1. Единая точка входа
Клиент обращается к одному адресу вместо нескольких микросервисов.

2. Reverse proxy
HTTP-запросы маршрутизируются в Auth, User, Chat и Voice Service.

3. WebSocket gateway
Одно клиентское WebSocket-соединение может использоваться для Chat и Voice, а Gateway маршрутизирует сообщения по назначению.

4. Пограничный уровень безопасности и инфраструктуры
Gateway выполняет JWT validation, CORS, logging, recovery, timeout и контроль соединений.

# Основные технологии
	Go;
	net/http;
	Gorilla Mux;
	Gorilla WebSocket;
	JWT (github.com/golang-jwt/jwt/v5);
	rs/cors;
	HTTP reverse proxy на базе net/http;
	WebSocket proxy;
	goroutines;
	channels;
	sync.Map;
	context;
	graceful shutdown.

По сути, в архитектуре Sozvon Gateway является фасадом всей backend-системы: клиент знает только Gateway, а Gateway уже скрывает от него расположение и внутренние интерфейсы Auth, User, Chat и Voice Service.