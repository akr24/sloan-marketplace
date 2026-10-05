# Build stage: Go 1.26, which whatsmeow requires (a local 1.25 install
# downloads the 1.26 toolchain automatically).
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

# Runtime stage: a static binary needs nothing but CA certs.
FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
USER app
COPY --from=build /out/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
