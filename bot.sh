#!/bin/bash

BUILD=build
APP=$BUILD/app
PID=$BUILD/.pid
LOG=$BUILD/log
DB=$BUILD/db

CMD=$1
case "$CMD" in

	"build")
		go build -o $APP -v
		;;

	"start")
		if [ -f $PID ]; then
			echo "Already running"
		else
			$APP -token=$2 -db=$DB -log=debug > $LOG 2>&1 &
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