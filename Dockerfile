FROM fedora
MAINTAINER Michail Kargakis <kargakis@protonmail.ch>

ENV GOPATH /go
ENV THYRA_STATIC /go/src/github.com/gothyra/thyra/static/

RUN dnf install -y git golang telnet
RUN go get github.com/gothyra/thyra

EXPOSE 4000

ENTRYPOINT /go/bin/thyra
