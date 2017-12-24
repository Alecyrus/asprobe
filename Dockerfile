FROM golang:latest

MAINTAINER Alecyrus "alecyrus@163.com"

WORKDIR $GOPATH/src/github.com/Alecyrus
ADD . $GOPATH/src/github.com/Alecyrus
RUN go build .

ENTRYPOINT ["./asprobe serve -n 50 "]
