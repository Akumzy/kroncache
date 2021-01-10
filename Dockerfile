FROM golang:1.15-alpine AS build_base

RUN apk add --no-cache git

# Set the Current Working Directory inside the container
WORKDIR /tmp/kroncache

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

# Unit tests
RUN CGO_ENABLED=0 go test -v

# Build the Go app
RUN go build .

# Start fresh from a smaller image
FROM alpine:latest

RUN apk add ca-certificates
WORKDIR /app
COPY --from=build_base /tmp/kroncache/kroncache /app/kroncache

RUN ls
# This container exposes port 5093 to the outside world
EXPOSE 5093

# Run the binary program produced by `go install`
ENTRYPOINT  ["/app/kroncache"]