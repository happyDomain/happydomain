FROM node:20-alpine as nodebuild

WORKDIR /go/src/git.happydns.org/happydomain

COPY ui/ ui/

RUN yarn config set network-timeout 100000 && \
    yarn --cwd ui install && \
    yarn --cwd ui --offline build


FROM golang:1-alpine as gobuild

RUN apk add --no-cache git

WORKDIR /go/src/git.happydns.org/happydomain

COPY --from=nodebuild /go/src/git.happydns.org/happydomain/ ./
COPY actions ./actions
COPY admin ./admin
COPY api ./api
COPY config ./config
COPY forms ./forms
COPY generators ./generators
COPY internal ./internal
COPY model ./model
COPY providers ./providers
COPY services ./services
COPY storage ./storage
COPY utils ./utils
COPY generate.go go.mod go.sum main.go ./

RUN sed -i '/npm run build/d' ui/assets.go && \
    go install github.com/swaggo/swag/cmd/swag@latest && \
    go generate -v ./... && \
    go build -v -ldflags '-w'


FROM alpine:3.18

EXPOSE 8081

ENTRYPOINT ["/usr/sbin/happydomain"]

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

COPY --from=gobuild /go/src/git.happydns.org/happydomain/happydomain /usr/sbin/happydomain
COPY hadmin.sh /usr/bin/hadmin
