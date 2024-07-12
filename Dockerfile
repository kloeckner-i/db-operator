FROM --platform=$BUILDPLATFORM registry.hub.docker.com/library/golang:1.22.2-alpine3.18 as builder


RUN apk update && apk upgrade && \
RUN apk update && apk upgrade && \
    apk add --no-cache bash build-base
    apk add --no-cache bash build-base
@@ -13,10 +13,13 @@ RUN go mod download
# build
# build
COPY . .
COPY . .


ARG GOARCH
RUN uname -a
RUN GOOS=linux GOARCH=$GOARCH CGO_ENABLED=0 go build -tags build -o /usr/local/bin/db-operator cmd/main.go

ARG TARGETARCH
RUN GOOS=linux GOARCH=$TARGETARCH CGO_ENABLED=0 go build -tags build -o /usr/local/bin/db-operator cmt/main.go


FROM registry.hub.docker.com/library/alpine:3.18
RUN uname -a
LABEL org.opencontainers.image.authors="Nikolai Rodionov<allanger@zohomail.com>"
LABEL org.opencontainers.image.authors="Nikolai Rodionov<allanger@zohomail.com>"


ENV USER_UID=1001
ENV USER_UID=1001