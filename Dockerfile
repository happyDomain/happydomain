FROM node:alpine as nodebuild

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
COPY admin ./admin
COPY api ./api
COPY config ./config
COPY forms ./forms
COPY generators ./generators
COPY model ./model
COPY services ./services
COPY sources ./sources
COPY storage ./storage
COPY utils ./utils
COPY generate.go go.mod go.sum main.go static.go ./

RUN sed -i '/yarn --cwd htdocs --offline build/d' static.go && \
    go get -d -v && \
    go generate -v && \
    go build -v


FROM alpine

EXPOSE 8081

CMD ["happydns"]

ENV HAPPYDNS_LEVELDB_PATH=/data/happydns.db

VOLUME /data

COPY --from=gobuild /go/src/git.happydns.org/happydns/happydns /usr/sbin/happydns
COPY hadmin.sh /usr/bin/hadmin
