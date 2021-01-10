FROM golang:1.15-alpine AS build_base

RUN apk add --no-cache git

# Set the Current Working Directory inside the container
WORKDIR /tmp/app

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

COPY --from=build_base /tmp/app/kroncache /app/kroncache

# This container exposes port 5093 to the outside world
EXPOSE 5093

RUN chmod +x /app/kroncache

# Run the binary program produced by `go install`
CMD ["/app/kroncache"]