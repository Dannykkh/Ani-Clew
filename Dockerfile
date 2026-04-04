FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /proxy ./cmd/proxy/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /proxy /usr/local/bin/proxy
EXPOSE 4000
ENTRYPOINT ["proxy"]
CMD ["--port", "4000"]
