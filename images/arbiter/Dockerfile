# syntax=docker/dockerfile:1

FROM golang:1.18-alpine
WORKDIR /app

COPY internal ./internal
COPY cmd ./cmd
COPY go.mod .
COPY go.sum .

RUN go build -o /arbiter cmd/arbiter/main.go cmd/arbiter/wire_gen.go

ENV PORT=:8888
EXPOSE 8888

CMD [ "/arbiter" ]
