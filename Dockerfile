FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/pswitch ./cmd/pswitch

FROM alpine:3.22

WORKDIR /data

COPY --from=builder /out/pswitch /usr/local/bin/pswitch

EXPOSE 8080

ENTRYPOINT ["pswitch"]
CMD ["--config", "/data/config.toml"]
