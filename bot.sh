#!/bin/bash

BUILD=build
APP=app
CONFIG=app.conf
PID=.pid
LOG=log

CMD=$1
case "$CMD" in

	"build")
		go build -o $BUILD/$APP -v
		;;
		
	"run")
	    cd $BUILD
		./$APP -config=$CONFIG
		;;

	"start")
	    cd $BUILD
		if [ -f $PID ]; then
			echo "Already running"
		else
			./$APP -config=$CONFIG > $LOG 2>&1 &
			echo $! > $PID
		fi
		;;

	"stop")
	    cd $BUILD
		if [ -f $PID ]; then
			kill `cat $PID`
			rm $PID
		else
			echo "Not running"
		fi
		;;
		
esac