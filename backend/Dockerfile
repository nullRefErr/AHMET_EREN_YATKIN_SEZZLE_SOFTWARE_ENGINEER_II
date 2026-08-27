# syntax=docker/dockerfile:1
ARG GO_VERSION=1.27

FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /src

# Dependencies are their own layer, so editing source does not re-download the module
# cache on every build.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

ARG VERSION=dev
# CGO_ENABLED=0 produces a static binary, which is what lets the final image be
# distroless. -trimpath keeps the build reproducible; -s -w drop the debug tables.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/server ./cmd/server

# No shell, no package manager, no root. Roughly 15 MB and close to nothing to attack.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/server /server
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
