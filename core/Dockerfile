FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main ./cmd/hezzl-test/main.go

FROM alpine:latest AS final
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/config ./config
EXPOSE 8082
RUN /bin/sh
CMD ["./main"]