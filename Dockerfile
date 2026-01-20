FROM node:24-alpine AS nodebuild

WORKDIR /go/src/git.happydns.org/happydomain

COPY web/ web/

RUN yarn config set network-timeout 100000 && \
    yarn --cwd web install && \
    yarn --cwd web --offline build


FROM golang:1-alpine AS gogenerator

WORKDIR /go/src/git.happydns.org/happydomain

COPY --from=nodebuild /go/src/git.happydns.org/happydomain/ ./
COPY cmd ./cmd
COPY tools ./tools
COPY internal ./internal
COPY model ./model
COPY providers ./providers
COPY services ./services
COPY generate.go ./

RUN sed -i '/npm run build/d;/npm run generate:api/d' web/assets.go web-admin/assets.go && \
    go install github.com/swaggo/swag/cmd/swag@latest && \
    go generate -v ./...


FROM node:24-alpine AS nodebuild-admin

WORKDIR /go/src/git.happydns.org/happydomain

COPY --from=gogenerator docs-admin/ docs-admin/
COPY web-admin/ web-admin/

RUN yarn config set network-timeout 100000 && \
    yarn --cwd web-admin install && \
    yarn --cwd web-admin --offline build


FROM golang:1-alpine AS gobuild

RUN apk add --no-cache git

WORKDIR /go/src/git.happydns.org/happydomain

COPY --from=nodebuild /go/src/git.happydns.org/happydomain/ ./
COPY --from=gogenerator /go/src/git.happydns.org/happydomain/providers/icons.go providers/icons.go
COPY --from=gogenerator /go/src/git.happydns.org/happydomain/services/icons.go services/icons.go
COPY --from=gogenerator /go/src/git.happydns.org/happydomain/web/src/lib/dns_rr.ts web/src/lib/dns_rr.ts
COPY --from=gogenerator /go/src/git.happydns.org/happydomain/internal/usecase/service_specs_dns_types.go internal/usecase/service_specs_dns_types.go
COPY --from=gogenerator /go/src/git.happydns.org/happydomain/docs/ docs/
COPY --from=gogenerator /go/src/git.happydns.org/happydomain/docs-admin/ docs-admin/
COPY cmd ./cmd
COPY tools ./tools
COPY internal ./internal
COPY model ./model
COPY providers ./providers
COPY services ./services
COPY generate.go go.mod go.sum ./

RUN go build -v -tags netgo,swagger,web -ldflags '-w' ./cmd/happyDomain/


FROM alpine:3.23

EXPOSE 8081

ENTRYPOINT ["/usr/sbin/happyDomain"]

HEALTHCHECK CMD curl --fail http://localhost:8081/api/version

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

COPY --from=gobuild /go/src/git.happydns.org/happydomain/happyDomain /usr/sbin/happyDomain
COPY hadmin.sh /usr/bin/hadmin
