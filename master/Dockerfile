FROM golang:1.9-alpine

RUN apk -U add git curl musl-dev gcc
RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.3.2/dep-linux-amd64 \
		&& chmod +x /usr/local/bin/dep

RUN mkdir -p $GOPATH/src/github.com/ugwis/online-compiler
WORKDIR $GOPATH/src/github.com/ugwis/online-compiler

COPY ./Gopkg.toml ./Gopkg.lock ./
RUN dep ensure -vendor-only

ADD . ./
RUN go build --ldflags '-s -w -linkmode external -extldflags -static' -o /main


FROM scratch
ADD ./languages.yaml ./languages.yaml
COPY --from=0 /main /main
CMD ["/main"]
