ARG GOLANG_VERSION=1.25.0
ARG ALPINE_VERSION=3.22

FROM golang:${GOLANG_VERSION}-alpine${ALPINE_VERSION} AS builder

# Install build tools
RUN apk add --no-cache make git

# Set build root
WORKDIR /build

# Copy go.work and go.work.sum (for multi-module builds)
COPY go.work go.work.sum ./

# -------------------------
# Copy shared package
WORKDIR /build/packages/shared
COPY packages/shared/go.mod packages/shared/go.sum ./
RUN go mod download
COPY packages/shared/ .

# -------------------------
# Copy auth package
WORKDIR /build/packages/auth
COPY packages/auth/go.mod packages/auth/go.sum ./
RUN go mod download
COPY packages/auth/ .

# -------------------------
# Copy db package
WORKDIR /build/packages/db
COPY packages/db/go.mod packages/db/go.sum ./
RUN go mod download
COPY packages/db/ .

# -------------------------
WORKDIR /build/packages/user
COPY packages/user/go.mod packages/user/go.sum Makefile ./
RUN go mod download
COPY packages/user/ .

# -------------------------
# Copy generated code + protos
WORKDIR /build/gen
COPY gen/ .

WORKDIR /build/proto
COPY proto/ .

# -------------------------
# Build the binary (using your Makefile)
ARG COMMIT_SHA
ARG EXPECTED_MIGRATION_TIMESTAMP
RUN --mount=type=cache,target=/root/.cache/go-build \
    make build COMMIT_SHA=${COMMIT_SHA} EXPECTED_MIGRATION_TIMESTAMP=${EXPECTED_MIGRATION_TIMESTAMP}

# -------------------------
# Final image
FROM alpine:${ALPINE_VERSION}

WORKDIR /app
COPY --from=builder /build/packages/auth/bin/auth-service ./auth-service



ENTRYPOINT ["./auth-service"]
