FROM golang:1.24

RUN apt-get update && apt-get install -y gcc libc6-dev sqlite3 libsqlite3-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1

RUN go build -o server ./cmd/api

EXPOSE 8080

CMD ["./server"]