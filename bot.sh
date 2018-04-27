#!/bin/bash

PATH=PATH:$HOME/Go/bin

RUNFILE=$HOME/.hikkabot
LOGFILE=$HOME/logs/hikkabot.log

archive_logs() {
    if [[ -f ${LOGFILE} ]]; then
        SUFFIX=`date +%F_%R`
        mv ${LOGFILE} "${LOGFILE}.${SUFFIX}"
    fi
}

start() {
    if [[ -f ${RUNFILE} ]]; then
        echo "Hikkabot instance already running, PID: `cat ${RUNFILE}`"
    else
        CONFIG=$1
        if [[ -n ${LOGFILE} ]]; then
            hikkabot -config=${CONFIG} 2>&1 > ${LOGFILE} &
            echo -e "PID=$!" > ${RUNFILE}
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
        archive_logs
        echo "OK"
    else
        echo "Hikkabot instance not running"
    fi
}

notify() {
    CONFIG=$1 CHAT=$2 NOTIFY=$4
    TEXT=`echo $3 | sed -r 's/\s+/%20/g;s/\./%2E/g'`
    TOKEN=`cat ${CONFIG} | jq -r ".token"`
    FORM="chat_id=${CHAT}&text=${TEXT}"
    if [[ ! ${NOTIFY} ]]; then
        FORM="${FORM}&disable_notification=true"
    fi

    curl -s -d ${FORM} -X POST https://api.telegram.org/bot${TOKEN}/sendMessage > /dev/null
}

check() {
    CONFIG=$1
    CHAT=$2
    if [[ -f ${RUNFILE} ]]; then
        source ${RUNFILE}
        kill -0 ${PID}
        if [[ $? -ne 0 ]]; then
            rm ${RUNFILE}
            notify ${CONFIG} ${CHAT} "Instance is not running. Restarting." 1
            archive_logs
            start $CONFIG
        else
            STATS=`ps -p ${PID} -o %cpu,%mem | tail -1`
            notify ${CONFIG} ${CHAT} "${STATS}"
        fi
    else
        notify ${CONFIG} ${CHAT} "Runfile not found." 1
    fi
}

case $1 in
    "start")
        start $2
        ;;

    "stop")
        stop
        ;;

    "check")
        check $2 $3
        ;;
esac