FROM golang:1.8
WORKDIR /usr/local/go/src/github.com/ugwis/online-compiler
RUN go get github.com/docker/docker/api/types
RUN go get github.com/docker/docker/api/types/container
RUN go get github.com/docker/docker/client
RUN go get github.com/gin-gonic/gin
RUN go get golang.org/x/net/context
ADD . ./
RUN go get ./...
RUN go build --ldflags '-s -w -linkmode external -extldflags -static' -o /main

FROM docker:dind
ADD . ./
COPY --from=0 /main /main
CMD ["/main"]
