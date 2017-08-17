if [ $# = 1 ];then
	docker rm $(docker ps -aq)
	docker pull ugwis/online-compiler
	docker run -it -d -v /var/run/docker.sock:/var/run/docker.sock -v /tmp/online-compiler:/tmp/online-compiler -v $(pwd):/app -w /app -p $1:3000 --privileged docker:dind sh start-app.sh 
else
	echo "Wrong arguments"
fi
