FROM ubuntu:xenial

RUN \
  apt-get update && apt-get upgrade -q -y && \
  apt-get install -y --no-install-recommends golang git make gcc libc-dev ca-certificates && \
  git clone --depth 1 --branch release/1.7 https://github.com/fairblock/go-fairblock && \
  (cd go-fairblock && make gfbc) && \
  cp go-fairblock/build/bin/gfbc /gfbc && \
  apt-get remove -y golang git make gcc libc-dev && apt autoremove -y && apt-get clean && \
  rm -rf /go-fairblock

EXPOSE 9565
EXPOSE 19565

ENTRYPOINT ["/gfbc"]
