#!/bin/bash

BUILD=build
APP=$BUILD/app
CONFIG=$BUILD/app.conf
PID=$BUILD/.pid
LOG=$BUILD/log

CMD=$1
case "$CMD" in

	"build")
		go build -o $APP -v
		;;
		
	"run")
		$APP -config=$CONFIG
		;;

	"start")
		if [ -f $PID ]; then
			echo "Already running"
		else
			$APP -config=$CONFIG 2>&1 > $LOG &
			echo $! > $PID
		fi
		;;

	"stop")
		if [ -f $PID ]; then
			kill `cat $PID`
			rm $PID
		else
			echo "Not running"
		fi
		;;
		
esac