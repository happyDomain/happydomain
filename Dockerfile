FROM node:14-alpine as nodebuild

WORKDIR /go/src/git.happydns.org/happydns

RUN apk --no-cache add python2 build-base

COPY ui/ ui/

RUN yarn config set network-timeout 100000 && \
    yarn --cwd ui install && \
    yarn --cwd ui --offline build


FROM golang:1-alpine as gobuild

RUN apk add --no-cache go-bindata

WORKDIR /go/src/git.happydns.org/happydns

COPY --from=nodebuild /go/src/git.happydns.org/happydns/ ./
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

RUN sed -i '/yarn --offline build/d' ui/assets.go && \
    go get -d -v && \
    go generate -v && \
    go build -v -ldflags '-w'


FROM alpine:3.15

EXPOSE 8081

ENTRYPOINT ["/usr/sbin/happydns"]

ENV HAPPYDNS_LEVELDB_PATH=/data/happydns.db

RUN apk add --no-cache \
        curl \
        jq \
    && \
    adduser --system --no-create-home --uid 15353 happydns && \
    mkdir /data && chown happydns /data
USER happydns
WORKDIR /data

VOLUME /data

COPY --from=gobuild /go/src/git.happydns.org/happydns/happydns /usr/sbin/happydns
COPY hadmin.sh /usr/bin/hadmin
