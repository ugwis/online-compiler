FROM golang:1.8
WORKDIR /usr/local/go/src/github.com/ugwis/online-compiler
ADD . ./
RUN go get ./... && go build --ldflags '-s -w -linkmode external -extldflags -static' -o /main

FROM scratch
COPY --from=0 /main /main
CMD ["/main"]
