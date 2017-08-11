FROM ubuntu:16.04
RUN apt-get update && apt-get install -y ruby \
	python \
	gcc \
	g++ \
	time \
	binutils \
	php \
	nodejs \
	npm && npm install express body-parser
