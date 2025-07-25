# SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

# Build stage
FROM golang:1.24-alpine AS build
ARG GOPROXY

WORKDIR /build

# Install build tools
RUN apk add --no-cache git make

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -o /build/xapp-hello-world ./cmd/xapp-hello-world

# Final stage
FROM alpine:3.15
WORKDIR /app

# Copy the binary from the build stage
COPY --from=build /build/xapp-hello-world .

# Set the entrypoint
ENTRYPOINT ["/app/xapp-hello-world"]
