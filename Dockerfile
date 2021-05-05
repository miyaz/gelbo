FROM golang:1.16-buster

WORKDIR /go/src/work

ADD . /go/src/work

RUN go mod download

CMD /bin/bash
