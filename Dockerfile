FROM golang:1.16 as build-env

ARG VERSION=0.0.0

# cache dependencies first
WORKDIR /code
COPY go.mod .
COPY go.sum .
RUN go mod download

# lastly copy source, any change in source will not break above cache
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -a -ldflags="-s -w -X finan/ms-order-management/pkg/service.version=${VERSION}" \
  -o app ./cmd/server

# runtime
FROM ubuntu:20.04

ENV ENABLE_DB=true

RUN apt-get update \
  && DEBIAN_FRONTEND="noninteractive" apt-get -y install tzdata ca-certificates --no-install-recommends \
  && ln -fs /usr/share/zoneinfo/Asia/Ho_Chi_Minh /etc/localtime \
  && rm -fr /var/lib/apt/lists/*

WORKDIR /code

COPY --from=build-env /code/app app

CMD ["/code/app"]

