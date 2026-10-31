# syntax=docker/dockerfile:1
FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/pokedex ./cmd/pokedex

# Distroless has no shell or package manager, and runs as an unprivileged user.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/pokedex /pokedex
EXPOSE 8080
# The binary probes itself, since the image has no curl.
HEALTHCHECK --interval=10s --timeout=4s --start-period=5s --retries=3 CMD ["/pokedex", "-healthcheck"]
ENTRYPOINT ["/pokedex"]
