FROM node:24-alpine AS web
WORKDIR /src
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.26-alpine AS api
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=web /src/dist ./web/dist
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=${VERSION}" -o /app .

# O Node fica na imagem final: o Claude Code roda nele (fase 3, T4).
FROM node:24-alpine
RUN apk add --no-cache git ripgrep libgcc libstdc++ \
 && npm install -g @anthropic-ai/claude-code \
 && npm cache clean --force
COPY --from=api /app /app
# Regras da IA (T6); as editadas em Ajustes ficam em /data/regras e valem por cima destas.
COPY regras/ /regras/
# Sem volume montado, o app ainda sobe (os dados ficam no container).
RUN mkdir -p /data && chown 1000:1000 /data
USER 1000:1000
ENV DATA_DIR=/data \
    REGRAS_DIR=/regras \
    HOME=/data/home \
    USE_BUILTIN_RIPGREP=0 \
    DISABLE_AUTOUPDATER=1
ENTRYPOINT ["/app"]
