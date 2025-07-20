FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY . .

WORKDIR /app/server
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/easyukey-server -trimpath -ldflags "-w -s -buildid=" .

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app/

COPY --from=builder /app/easyukey-server .

EXPOSE 8888

CMD ["./easyukey-server"]
