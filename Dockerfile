FROM golang:1.21.13-bookworm

WORKDIR /go/src/app
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
RUN go get -d -v ./...
COPY ./main.go ./main.go
COPY ./server ./server

RUN GOOS=$(uname | tr '[:upper:]' '[:lower:]') GOARCH=amd64 go build -ldflags "-s -w" -o speech-to-text

FROM debian:bookworm-20250520

WORKDIR /go/src/app

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=0 /go/src/app/speech-to-text /
RUN useradd -ms /bin/bash www && \
    chown -R www:www /go/src/app && \
    chmod 550 /speech-to-text

USER www

CMD ["/speech-to-text"]
