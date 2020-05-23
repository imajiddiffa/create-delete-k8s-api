########### First Stage, Build Golang App to executable binary ############
FROM golang:1.11.5-alpine3.9 AS builder

# Install tools required to build the project
# We will need to run `docker build --no-cache .` to update those dependencies
RUN apk add --no-cache git curl

# Gopkg.toml and Gopkg.lock lists project dependencies
# These layers will only be re-built when Gopkg files are updated
COPY . /go/src/github.com/imajiddiffa/create-delete-k8s-api
WORKDIR /go/src/github.com/imajiddiffa/create-delete-k8s-api

# Install library dependencies
#RUN dep ensure -vendor-only
RUN set -x && \
    go get github.com/golang/dep/cmd/dep && \
    dep ensure -v

# Set config apps for production or development
# RUN sh deployment/scripts/configapp.sh

# Copy all project and build it
# This layer will be rebuilt when ever a file has changed in the project directory
RUN env CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o create-delete-k8s-api -v .

########### Second Stage, Get CA-Certificate And TimeZone Info ##############
FROM alpine:latest as alpine
RUN apk --no-cache add tzdata ca-certificates curl

# install doctl & kubectl
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
RUN chmod +x ./kubectl
RUN mv ./kubectl /usr/local/bin

WORKDIR /root
RUN mkdir -p templates

COPY --from=builder /go/src/github.com/imajiddiffa/create-delete-k8s-api/create-delete-k8s-api .
COPY --from=builder /go/src/github.com/imajiddiffa/create-delete-k8s-api/templates/pod.yml templates/

## tell how to run this container
CMD ["./create-delete-k8s-api"]

EXPOSE 10000
