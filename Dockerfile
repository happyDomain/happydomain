FROM node:lts-alpine as nodebuild

WORKDIR /go/src/git.happydns.org/happydns

RUN apk --no-cache add python2 build-base

COPY htdocs/ htdocs/

RUN yarn config set network-timeout 100000
RUN yarn --cwd htdocs install
RUN yarn --cwd htdocs --offline build


FROM golang:alpine as gobuild

RUN apk add --no-cache go-bindata

WORKDIR /go/src/git.happydns.org/happydns

COPY --from=nodebuild /go/src/git.happydns.org/happydns/ ./
COPY actions ./actions
COPY admin ./admin
COPY api ./api
COPY config ./config
COPY forms ./forms
COPY generators ./generators
COPY model ./model
COPY providers ./providers
COPY services ./services
COPY storage ./storage
COPY utils ./utils
COPY generate.go go.mod go.sum main.go static.go ./

RUN sed -i '/yarn --cwd htdocs --offline build/d' static.go && \
    go get -d -v && \
    go generate -v && \
    go build -v -ldflags '-w'


FROM alpine

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
