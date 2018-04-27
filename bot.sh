#!/bin/bash

RUNFILE=$HOME/.hikkabot

start() {
    if [[ -f ${RUNFILE} ]]; then
        echo "Hikkabot instance already running, PID: `cat ${RUNFILE}`"
    else
        CONFIG=$1 LOGFILE=`realpath $2`
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

notify() {
    CONFIG=$1 CHAT=$2 TEXT=$3 NOTIFY=$4
    TOKEN=`cat ${CONFIG} | jq -r ".token"`
    FORM="chat_id=${CHAT}&text=#health%0A${TEXT}"
    if [[ ! ${NOTIFY} ]]; then
        FORM="${FORM}&disable_notifications=true"
    fi

    curl -s -d ${FORM} -H "Content-Type: application/x-www-form-urlencoded" -X POST https://api.telegram.org/bot${TOKEN}/sendMessage > /dev/null
}

check() {
    CONFIG=$1
    CHAT=$2
    if [[ -f ${RUNFILE} ]]; then
        source ${RUNFILE}
        kill -0 ${PID}
        if [[ $? -ne 0 ]]; then
            rm ${RUNFILE}
            notify ${CONFIG} ${CHAT} "Instance is not running." 1
        else
            STATS=`ps -p ${PID} -o %cpu,%mem | tail -1`
            notify ${CONFIG} ${CHAT} ${STATS}
        fi
    else
        notify ${CONFIG} ${CHAT} "Runfile not found." 1
    fi
}

case $1 in
    "start")
        start $2 $3
        ;;

    "stop")
        stop
        ;;

    "check")
        check $2 $3
        ;;
esac