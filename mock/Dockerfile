FROM golang:1.15.12-alpine3.13 as builder

WORKDIR /go/src/kci/mock

# to reduce docker build time download dependency first before building
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY sqladmin.go .
RUN go build -o /usr/local/bin/sqladmin ./

EXPOSE 8080/tcp
CMD ["sqladmin"]
