docker rm $(docker ps -aq)
docker run -it -d -v /var/run/docker.sock:/var/run/docker.sock -v $(pwd):/app -w /app -p 3000:3000 --privileged docker:dind sh start-app.sh 
