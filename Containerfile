FROM docker.io/golang:1.25-alpine3.22 AS buildStage

WORKDIR /root/bunsamosa-bot

RUN apk add --no-cache --update git build-base
ENV GOPATH /root/bunsamosa-bot

COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY database/ ./database
COPY globals/ ./globals
COPY handlers/ ./handlers
COPY main.go ./

RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags='-s -w' -o bunsamosa-bot

FROM docker.io/alpine:3.22

RUN mkdir -p /root/bunsamosa-bot/logs
RUN apk --no-cache add libc6-compat libgcc libstdc++
WORKDIR /root/bunsamosa-bot
COPY --from=buildStage /root/bunsamosa-bot/bunsamosa-bot /opt/bunsamosa-bot

ENV JSON_LOG_DIR="/root/bunsamosa-bot/logs"

ENTRYPOINT ["/opt/bunsamosa-bot"]
