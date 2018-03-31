#!/bin/bash

PIDFILE=$HOME/.hikkabot.pid

start() {
    if [[ -f ${PIDFILE} ]]; then
        echo "Hikkabot instance already running, PID: `cat ${PIDFILE}`"
    else
        CONFIG=$1
        LOGFILE=$2
        if [[ -n ${LOGFILE} ]]; then
            hikkabot -config=${CONFIG} 2>&1 > ${LOGFILE} &
            echo $! > ${PIDFILE}
        else
            hikkabot -config=${CONFIG}
        fi
    fi
}

stop() {
    if [[ ! -f ${PIDFILE} ]]; then
        kill `cat ${PIDFILE}`
    else
        echo "Hikkabot instance not running"
    fi
}

case $1 in
    "start")
        start $2 $3
        ;;

    "stop")
        stop
        ;;
esac