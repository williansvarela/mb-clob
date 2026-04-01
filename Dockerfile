# Build stage
FROM golang:1.26-alpine AS builder

ARG CGO_ENABLED=0
ARG GOOS=linux

WORKDIR /build

COPY go.mod ./

RUN go mod download && go mod verify

COPY . .

# Build the application
RUN CGO_ENABLED=${CGO_ENABLED} GOOS=${GOOS} \
    go build \
    -o mb-clob \
    ./cmd/main.go


# Stage 2: Runtime stage
FROM gcr.io/distroless/static-debian12 AS runtime

COPY --from=builder /build/mb-clob /mb-clob

# Expose the API port
EXPOSE 8080

# Use distroless nonroot user (UID 65532)
USER nonroot:nonroot

ENTRYPOINT ["/mb-clob"]