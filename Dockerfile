FROM golang:1.25-alpine AS build
WORKDIR /src

# Cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/WarhoopAll/wowchat/internal/version.Version=${VERSION}" \
    -o /out/wowchat ./cmd/wowchat

# --- Runtime stage ---
FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=build /out/wowchat /app/wowchat

CMD ["./wowchat"]
