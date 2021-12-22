FROM golang:alpine AS dev

WORKDIR /root

RUN apk add cmake make g++ eudev-dev p8-platform-dev linux-headers

ENV LIBCEC_VERSION=6.0.2

ADD https://github.com/Pulse-Eight/libcec/archive/libcec-${LIBCEC_VERSION}.tar.gz .

RUN rm -rf libcec \
  && tar xf libcec-${LIBCEC_VERSION}.tar.gz \
  && mv libcec-libcec-${LIBCEC_VERSION} libcec \
  \
  && mkdir libcec/build \
  && cd libcec/build \
  \
  && cmake -DCMAKE_BUILD_TYPE=Release \
       -DBUILD_SHARED_LIBS=1 \
       -DCMAKE_INSTALL_PREFIX=/usr \
       -DHAVE_LINUX_API=1 \
       .. \
  \
  && make -j4 \
  \
  && make install

WORKDIR /go/src/cec2mqtt

FROM dev AS builder

COPY . ./

RUN apk add git

RUN go get -d -v .
RUN go build -v -o cec2mqtt

FROM alpine

RUN apk add p8-platform eudev

COPY --from=builder /usr/lib/libcec.so* /usr/lib/
COPY --from=builder /go/src/cec2mqtt/cec2mqtt /usr/bin

VOLUME /data/cec2mqtt

CMD ["/usr/bin/cec2mqtt"]
