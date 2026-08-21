FROM golang:1.23-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bin/ingest ./cmd/ingest

FROM alpine:3.20

RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=build /app/bin/ingest /app/bin/ingest
COPY keys/public.pem /app/keys/public.pem

ENV INGEST_PUBLIC_KEY_FILE=/app/keys/public.pem

EXPOSE 8080
CMD ["/app/bin/ingest"]
