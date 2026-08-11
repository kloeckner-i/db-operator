FROM --platform=$BUILDPLATFORM registry.hub.docker.com/library/golang:1.26.5-alpine3.24 AS builder

ARG OPERATOR_VERSION=1.0.0-dev
WORKDIR /opt/db-operator

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
ARG TARGETARCH
RUN GOOS=linux GOARCH=$TARGETARCH CGO_ENABLED=0 \
  go build \
  -ldflags="-X \"github.com/db-operator/db-operator/v2/internal/helpers/common.OperatorVersion=$OPERATOR_VERSION\"" \
  -tags build -o /usr/local/bin/db-operator cmd/main.go


FROM gcr.io/distroless/static
LABEL org.opencontainers.image.authors="Nikolai Rodionov<allanger@posteo.com>"
COPY --from=builder /usr/local/bin/db-operator /usr/local/bin/db-operator
USER 1001
ENTRYPOINT ["/usr/local/bin/db-operator"]
