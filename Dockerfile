FROM golang:1.15-buster

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    curl \
    dnsutils \
    git \
    gnupg \
    gnupg1 \
    gnupg2 \
    htop \
    iputils-ping \
    less \
    locales \
    lsof \
    mc \
    mtr \
    netcat \
    procps \
    tar \
    telnet \
    unzip \
    vim \
    wget \
    zip \
  && rm -rf /var/cache/apt/archives/*

WORKDIR /go/src/app
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
RUN go get -d -v ./...
COPY ./main.go ./main.go
COPY ./server ./server

RUN GOOS=`uname| tr '[:upper:]' '[:lower:]'` GOARCH=amd64 go build -o build

RUN useradd -ms /bin/bash www
RUN chown -R www:www /go/src/app
USER www

CMD ["./build"]
