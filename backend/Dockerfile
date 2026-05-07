FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION=dev
ARG REVISION=unknown
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${VERSION} -X main.revision=${REVISION}" -o workflow-command backend/services/workflow-command/cmd/server/main.go

FROM alpine:3.22

ARG VERSION=dev
ARG REVISION=unknown

WORKDIR /app

RUN addgroup -S goflow && adduser -S -G goflow goflow
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/workflow-command .
USER goflow

EXPOSE 8080

LABEL org.opencontainers.image.title="GoFlow Command Service" \
      org.opencontainers.image.description="Command API service for GoFlow" \
      org.opencontainers.image.source="https://github.com/goflow/goflow" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}"

CMD ["./workflow-command"]
