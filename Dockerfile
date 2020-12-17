FROM golang:1.15

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...
RUN GOOS=`uname| tr '[:upper:]' '[:lower:]'` GOARCH=amd64 go build -o build

CMD ["./build"]
