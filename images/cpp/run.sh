#!/bin/sh
timeout 3 ./main
status_code=$?
if [ ${status_code} -ne 0 ];then
	echo "Exit Code: ${status_code}"
	if [ ${status_code} -eq 139 ];then
		gdb main core --batch
	fi
fi
