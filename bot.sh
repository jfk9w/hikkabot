#!/bin/bash

RUNFILE=$HOME/.hikkabot

start() {
    if [[ -f ${RUNFILE} ]]; then
        echo "Hikkabot instance already running, PID: `cat ${RUNFILE}`"
    else
        CONFIG=$1
        LOGFILE=$2
        if [[ -n ${LOGFILE} ]]; then
            hikkabot -config=${CONFIG} 2>&1 > ${LOGFILE} &
            echo -e "PID=$!\nLOGFILE=${LOGFILE}" > ${RUNFILE}
        else
            hikkabot -config=${CONFIG}
        fi
    fi
}

stop() {
    if [[ -f ${RUNFILE} ]]; then
        source ${RUNFILE}
        kill ${PID}
        echo "Waiting for Hikkabot instance death, PID: ${PID}"
        tail -f ${LOGFILE} | while read LOGLINE; do
            [[ "${LOGLINE}" == *"MAIN exit"* ]] && pkill -P $$ tail
        done
        rm ${RUNFILE}
        echo "OK"
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