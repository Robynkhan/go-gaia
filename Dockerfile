# Build Gfbc in a stock Go builder container
FROM golang:1.9-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

ADD . /go-fairblock
RUN cd /go-fairblock && make gfbc

# Pull Gfbc into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go-fairblock/build/bin/gfbc /usr/local/bin/

EXPOSE 9565 8546 19565 19565/udp 30304/udp
ENTRYPOINT ["gfbc"]
