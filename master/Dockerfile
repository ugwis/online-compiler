FROM docker:dind
WORKDIR /
ADD . .
RUN apk update && \ 
	apk add nodejs nodejs-npm && \
	npm update && \
	npm install express body-parser
CMD ["node", "app.js"]
