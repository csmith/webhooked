FROM golang:1.24.6 AS build
WORKDIR /go/src/app
COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    set -eux; \
    CGO_ENABLED=0 GO111MODULE=on go install .; \
    go run github.com/google/go-licenses@latest save ./... --save_path=/notices; \
    mkdir -p /mounts/data;

FROM ghcr.io/greboid/dockerbase/nonroot:1.20250803.0
COPY --from=build /go/bin/webhooked /webhooked
COPY --from=build /notices /notices
ENTRYPOINT ["/webhooked"]