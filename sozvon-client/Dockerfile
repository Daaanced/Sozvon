# Sozvon/sozvon-client/Dockerfile
# ─── Stage 1: build React ─────────────────────────────────────────────────────
FROM node:26-alpine AS builder

# Объявляем аргументы, которые придут из docker-compose
ARG VITE_API_URL
ARG VITE_WS_URL
ARG DOMAIN

# Превращаем их в ENV переменные, чтобы они были доступны Vite во время сборки
ENV VITE_API_URL=$VITE_API_URL
ENV VITE_WS_URL=$VITE_WS_URL
ENV DOMAIN=$DOMAIN

WORKDIR /app

COPY package.json package-lock.json* yarn.lock* ./
RUN npm install --prefer-offline

COPY . .
RUN npm run build

# ─── Stage 2: serve via nginx ─────────────────────────────────────────────────
FROM nginx:1.27-alpine

RUN rm /etc/nginx/conf.d/default.conf

COPY --from=builder /app/dist /usr/share/nginx/html

RUN printf 'server {\n\
    listen 3000;\n\
    root /usr/share/nginx/html;\n\
    index index.html;\n\
    location / {\n\
        try_files $uri $uri/ /index.html;\n\
    }\n\
}\n' > /etc/nginx/conf.d/spa.conf

EXPOSE 3000

CMD ["nginx", "-g", "daemon off;"]