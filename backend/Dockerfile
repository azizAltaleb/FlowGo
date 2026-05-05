FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION=dev
ARG REVISION=unknown
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${VERSION} -X main.revision=${REVISION}" -o workflow-command backend/services/workflow-command/cmd/server/main.go

FROM alpine:3.22

WORKDIR /app

RUN addgroup -S workflowsa && adduser -S -G workflowsa workflowsa
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/workflow-command .
USER workflowsa

EXPOSE 8080

LABEL org.opencontainers.image.title="Workflowsa Command Service" \
      org.opencontainers.image.description="Command API service for Workflowsa" \
      org.opencontainers.image.source="https://github.com/workflowsa/workflowsa" \
      org.opencontainers.image.licenses="MIT"

CMD ["./workflow-command"]
