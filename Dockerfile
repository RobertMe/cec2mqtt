ARG PLATFORM=default

FROM golang:alpine AS dev
ARG PLATFORM

WORKDIR /root

RUN apk add cmake make g++ eudev-dev p8-platform-dev

ENV LIBCEC_VERSION=4.0.4

COPY docker/install-libcec.base docker/install-libcec.${PLATFORM} ./

ADD https://github.com/Pulse-Eight/libcec/archive/libcec-${LIBCEC_VERSION}.tar.gz .

RUN ./install-libcec.$PLATFORM

WORKDIR /go/src/cec2mqtt
COPY . .

FROM dev AS builder

RUN go get -d -v .
RUN go build -v -o cec2mqtt

FROM alpine
ARG PLATFORM

RUN apk add p8-platform
RUN if [ "$PLATFORM" == "rpi" ] ; then apk add raspberrypi-libs ; fi

COPY --from=builder /usr/lib/libcec.so* /usr/lib/
COPY --from=builder /go/src/cec2mqtt/cec2mqtt /usr/bin

CMD ["/usr/bin/cec2mqtt"]
