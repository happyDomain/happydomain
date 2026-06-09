FROM golang:1-alpine AS gogenerator

WORKDIR /src

# First download dependancies
COPY go.mod go.sum ./

# renovate: datasource=go depName=github.com/swaggo/swag
ARG SWAG_VERSION=v1.16.6

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && \
    go install github.com/swaggo/swag/cmd/swag@${SWAG_VERSION}

# Generate go code
COPY checkers ./checkers
COPY cmd ./cmd
COPY internal ./internal
COPY model ./model
COPY pkg ./pkg
COPY providers ./providers
COPY services ./services
COPY tools ./tools
COPY generate.go ./

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    mkdir -p web/src/lib/services && \
    go generate ./...


FROM node:24-alpine AS nodebuild

WORKDIR /src

COPY --from=gogenerator /src/docs/ docs/
COPY web/ web/
COPY --from=gogenerator /src/web/src/lib/dns_rr.ts web/src/lib/dns_rr.ts
COPY --from=gogenerator /src/web/src/lib/services_specs.ts web/src/lib/services_specs.ts
COPY --from=gogenerator /src/web/src/lib/services/caa-issuers.json web/src/lib/services/caa-issuers.json

RUN --mount=type=cache,target=/usr/local/share/.cache/yarn \
    yarn --cwd web --network-timeout 100000 install && \
    yarn --cwd web --offline generate:api && \
    yarn --cwd web --offline build


COPY --from=gogenerator /src/docs-admin/ docs-admin/
COPY web-admin/ web-admin/

RUN --mount=type=cache,target=/usr/local/share/.cache/yarn \
    yarn --cwd web-admin --network-timeout 100000 install && \
    yarn --cwd web-admin --offline generate:api && \
    yarn --cwd web-admin --offline build


FROM golang:1-alpine AS gobuild

RUN apk add --no-cache git

WORKDIR /src

COPY --from=nodebuild /src/web/build/ web/build/
COPY --from=nodebuild /src/web-admin/build/ web-admin/build/
COPY web/*.go web/
COPY web-admin/*.go web-admin/
COPY --from=gogenerator /src/providers/icons.go providers/icons.go
COPY --from=gogenerator /src/services/icons.go services/icons.go
COPY --from=gogenerator /src/web/src/lib/dns_rr.ts web/src/lib/dns_rr.ts
COPY --from=gogenerator /src/internal/usecase/service_specs_dns_types.go internal/usecase/service_specs_dns_types.go
COPY --from=gogenerator /src/docs/ docs/
COPY --from=gogenerator /src/docs-admin/ docs-admin/
COPY checkers ./checkers
COPY cmd ./cmd
COPY internal ./internal
COPY model ./model
COPY pkg ./pkg
COPY providers ./providers
COPY services ./services
COPY tools ./tools
COPY generate.go go.mod go.sum ./

ARG VERSION=dev

RUN --mount=type=cache,target=/go/pkg/mod \
    go build -tags netgo,swagger,web -ldflags "-w -X main.Version=${VERSION}" ./cmd/happyDomain/


FROM alpine:3.24

ARG VERSION=dev

LABEL org.opencontainers.image.title="happyDomain" \
      org.opencontainers.image.description="Making DNS simple for everyone" \
      org.opencontainers.image.url="https://happydomain.org" \
      org.opencontainers.image.source="https://git.happydns.org/happydomain" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.licenses="AGPL-3.0-or-later"

EXPOSE 8081

ENTRYPOINT ["/usr/sbin/happyDomain"]

HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD curl --fail http://localhost:8081/api/version

ENV HAPPYDOMAIN_LEVELDB_PATH=/data/happydomain.db

RUN apk add --no-cache \
        curl \
        jq \
    && \
    adduser --system --no-create-home --uid 15353 happydomain && \
    mkdir /data && chown happydomain /data
USER happydomain
WORKDIR /data

VOLUME /data

COPY --from=gobuild /src/happyDomain /usr/sbin/happyDomain
COPY hadmin.sh /usr/bin/hadmin
