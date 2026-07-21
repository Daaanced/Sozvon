# Sozvon

> Современный desktop-мессенджер с микросервисной архитектурой, поддержкой текстовых сообщений, голосовых каналов и обмена файлами.

![Go](https://img.shields.io/badge/Go-1.26-blue?logo=go)
![React](https://img.shields.io/badge/React-19-61DAFB?logo=react)
![TypeScript](https://img.shields.io/badge/TypeScript-5-blue?logo=typescript)
![Electron](https://img.shields.io/badge/Electron-Latest-47848F?logo=electron)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-blue?logo=postgresql)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)

---

# О проекте

**Sozvon** — это клиент-серверный мессенджер, разработанный как учебный и портфолио-проект с использованием современной микросервисной архитектуры.

Основные возможности:

- 💬 личные и групповые чаты
- 🔊 голосовые комнаты (WebRTC)
- 📁 отправка файлов
- 🖼️ редактирование профиля (изменение автара, имени, личной информации)
- 🔐 JWT-аутентификация
- ⚡ WebSocket для обмена сообщениями
- 🐳 полное развёртывание через Docker Compose

---

# Архитектура

```
						   +--------------------+
						   |  React + Electron  |
						   +---------+----------+
									 |
								HTTP / WS
									 |
                    		   +-----v-----+
                               |  Gateway  |
                               +---+---+---+
                       		   |   |   |   |
                   		REST   |   |   |   |
		+---------------------+|   |   |   |+---------------------+
        |                   +-----+|   |+-----+                   |
        |                   |                 |                   |
+-------v------+   +--------v------+   +------v-------+   +-------v-------+  
| Auth Service |   | User Service  |   | Chat Service |   | Voice Service |
+--------------+   +---------------+   +--------------+   | WebRTC + WS   |
        |                  |                  |           +---------------+
	PostgreSQL         PostgreSQL         PostgreSQL

```
# Используемые технологии

## Backend

- Go
- Gorilla Mux
- PostgreSQL
- JWT
- WebSocket
- WebRTC (Pion)
- Docker

## Frontend

- React
- TypeScript
- Vite
- Electron
- React Router

## Infrastructure

- Docker Compose
- Nginx
- GitHub Actions (CI/CD)

---

# Структура проекта

```
Sozvon/

├── Gateway/
│
├── Services/
│   ├── Auth_Service/
│   ├── User_Service/
│   ├── Chat_Service/
│   └── Voice_Service/
│
├── sozvon-client/
│
├── nginx/
│
├── docker-compose.yml
├── docker-compose.dev.yml
├── docker-compose.prod.yml
└── README.md
```

---

# Микросервисы

## Gateway

Единая точка входа.

Отвечает за:

- маршрутизацию запросов
- проксирование WebSocket
- работу API

---

## Auth Service

Отвечает за:

- регистрацию
- авторизацию
- JWT-токены

Использует собственную БД PostgreSQL.

---

## User Service

Отвечает за:

- профиль пользователя
- аватар
- поиск пользователей

Использует собственную БД PostgreSQL.

---

## Chat Service

Отвечает за:

- создание чатов
- сообщения
- непрочитанные сообщения
- отправку файлов

Использует собственную БД PostgreSQL.

---

## Voice Service

Отвечает за:

- голосовые комнаты
- WebRTC-соединения
- сигнализацию через WebSocket

---

# Запуск проекта

## Клонирование

```bash
git clone https://github.com/Daaanced/Sozvon.git

cd Sozvon
```

---

## Настройка

Изменить файл

```
.env
```

например:

```env
LOCAL_IP=127.0.0.1
GLOBAL_IP=
DOMAIN=

```

---

## Запуск

Разработка

```bash
docker compose -f docker-compose.dev.yml up --build
```

---

# Порты

| Сервис | Порт |
|---------|------|
| Gateway | 8080 |
| Auth Service | 8082 |
| User Service | 8083 |
| Chat Service | 8084 |
| Client (Vite) | 3000 |

---

# Возможности

- Регистрация
- Авторизация
- Личные и групповые сообщения
- История сообщений
- Непрочитанные сообщения
- Загрузка файлов
- Изменение профиля, в том числе аватара
- Голосовые комнаты
- Docker-развёртывание
- Nginx Reverse Proxy
- HTTPS

---

# CI/CD

Пайплайн развёртывания:

```
git push

↓

GitHub Actions

↓

Сборка Docker-образов

↓

Публикация образов

↓

Подключение к серверу

↓

docker compose pull

↓

docker compose up -d
```

---

# Планы развития

- Доработка дизайна главных страниц
- Расширение настроек (устройства ввода/вывода, прочее)
- Видеозвонки
- Демонстрация экрана
- Работа с чатами (приглашение и удаление участников чата, удаление и изменения чата)
- Приложение (Desktop версия)

---

# Скриншоты

Будут добавлены после завершения интерфейса.

---

# Лицензия

Проект создан в образовательных целях.