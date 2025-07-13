FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR /app/cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata wget
RUN adduser -D -g '' appuser
WORKDIR /app
COPY --from=builder /app/cmd/server/main .
RUN chmod +x main
RUN chown appuser:appuser main
COPY .env .
COPY mride-51861-firebase-adminsdk-fbsvc-08c24a73d0.json . 
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1
ENTRYPOINT ["./main"]