FROM node:22-alpine AS build
WORKDIR /app
ARG VITE_API_BASE_URL=/api
ARG VITE_REQUIRE_PASSWORD=false
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
ENV VITE_REQUIRE_PASSWORD=${VITE_REQUIRE_PASSWORD}
COPY package.json package-lock.json* ./
RUN npm install
COPY . .
RUN npm run build

FROM caddy:2.8-alpine
COPY --from=build /app/dist /usr/share/caddy
COPY Caddyfile /etc/caddy/Caddyfile
EXPOSE 80
