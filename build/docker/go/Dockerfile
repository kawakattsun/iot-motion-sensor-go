FROM golang:1.13.4-alpine3.10

RUN apk add --no-cache \
        git \
        make \
        bash \
        gcc \
        libc-dev \
        openssl \
        curl

WORKDIR /go/src/${OWNER_NAME}/${PROJECT_NAME}

RUN go get -u \
    golang.org/x/lint/golint

COPY go.mod .
COPY go.sum .

RUN go mod download
