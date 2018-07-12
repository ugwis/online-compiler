#!/bin/bash
mkfifo gcc
gcc -ggdb -o ./main ./main.c > gcc
status_code=$?
(cat < gcc) | awk '{print "gcc:"$0 > "/dev/stdout";fflush()'&
if [ ${status_code} -ne 0 ];then
	echo "Exit Code: ${status_code}"
	exit 0
fi
mkfifo stdout
mkfifo stderr
(cat < stdout) | awk '{print "stdout:"$0 > "/dev/stdout";fflush()}'&
(cat < stderr) | awk '{print "stderr:"$0 > "/dev/stdout";fflush()}'&
timeout 3 ./main 1> stdout 2> stderr
status_code=$?
rm stdout&
rm stderr&
if [ ${status_code} -ne 0 ];then
	echo "Exit Code: ${status_code}"
	if [ ${status_code} -eq 139 ];then
		gdb main core --batch | awk '{print "gdb:$0";fflush()}'
	fi
	exit 0
fi
