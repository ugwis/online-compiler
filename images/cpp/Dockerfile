FROM alpine:3.6
RUN apk add --no-cache coreutils g++ gdb musl-dev && \
	g++ -std=c++0x $(find . | grep stdc++.h)
ADD . /
