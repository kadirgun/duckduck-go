FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /duckduckgo-server ./cmd/duckduckgo-server
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /duckduckgo-cli ./cmd/duckduckgo-cli

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /duckduckgo-server /duckduckgo-server
COPY --from=builder /duckduckgo-cli /duckduckgo-cli

EXPOSE 8080

ENTRYPOINT ["/duckduckgo-server"]
