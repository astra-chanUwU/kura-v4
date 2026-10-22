FROM node:24.22.1-alpine AS frontend
RUN npm install --global pnpm@12.15.0
WORKDIR /src
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY frontend/package.json frontend/package.json
RUN pnpm install --frozen-lockfile
COPY frontend frontend
RUN pnpm --filter @kura/frontend build

FROM golang:1.27.1-alpine AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd cmd
COPY internal internal
COPY --from=frontend /src/internal/webui/dist internal/webui/dist
RUN CGO_ENABLED=0 go build -trimpath -o /out/kura ./cmd/kura

FROM alpine:3.23
RUN apk add --no-cache ca-certificates ffmpeg \
    && adduser -D -u 10001 kura \
    && mkdir -p /data/media && chown -R kura:kura /data
COPY --from=backend /out/kura /usr/local/bin/kura
USER kura
EXPOSE 8094
ENTRYPOINT ["/usr/local/bin/kura"]
CMD ["serve"]
