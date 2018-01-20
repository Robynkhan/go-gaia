FROM alpine:3.5

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates && \
  git clone --depth 1 https://github.com/fairblock/go-fairblock && \
  (cd go-fairblock && make gfbc) && \
  cp go-fairblock/build/bin/gfbc /gfbc && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /go-fairblock && rm -rf /var/cache/apk/*

EXPOSE 9565
EXPOSE 19565

ENTRYPOINT ["/gfbc"]
