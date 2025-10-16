FROM docker.io/golang:1.25-alpine3.22 AS buildStage

WORKDIR /root/bunsamosa-bot

COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY database/ ./database
COPY globals/ ./globals
COPY handlers/ ./handlers
COPY main.go ./

RUN CGO_ENABLED=1 go build -trimpath -ldflags='-s -w' -o bunsamosa-bot

FROM docker.io/alpine:3.22

RUN mkdir -p /root/bunsamosa-bot/logs
WORKDIR /root/bunsamosa-bot
COPY --from=buildStage /root/bunsamosa-bot/bunsamosa-bot /opt/bunsamosa-bot

ENV JSON_LOG_DIR="/root/bunsamosa-bot/logs"

ENTRYPOINT ["/opt/bunsamosa-bot"]
