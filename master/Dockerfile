FROM golang:1.8
WORKDIR /usr/local/go/src/github.com/ugwis/online-compiler
RUN go get -u github.com/golang/dep/cmd/dep
ENV GOPATH /usr/local/go/
RUN dep init
ADD . ./
RUN go build --ldflags '-s -w -linkmode external -extldflags -static' -o /main

FROM docker:dind
ADD . ./
COPY --from=0 /main /main
CMD ["/main"]
