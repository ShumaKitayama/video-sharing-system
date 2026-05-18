FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY migrations ./migrations
COPY --from=build /out/api /usr/local/bin/api
ENV UPLOAD_DIR=/app/uploads
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
