#!/bin/bash

PATH=$PATH:$HOME/Go/bin

RUNFILE=$HOME/.hikkabot
LOGDIR=$HOME/logs/hikkabot/
PACKAGE=github.com/jfk9w-go/hikkabot

archive_logs() {
    if [[ -f ${LOGDIR} ]]; then
        DIR=${LOGDIR}/`date +%F_%R`
        mkdir ${DIR}
        mv ${LOGDIR}/*.log ${DIR}
    fi
}

start() {
    if [[ -f ${RUNFILE} ]]; then
        echo "Hikkabot instance already running, PID: `cat ${RUNFILE}`"
    else
        CONFIG=$1
        TOKEN=`cat ${CONFIG} | jq -r ".token"`
        env TOKEN=${TOKEN} LOGCFG=${CONFIG} hikkabot 2>&1 > ${LOGDIR}/main.log &
        echo -e "PID=$!" > ${RUNFILE}
    fi
}

stop() {
    if [[ -f ${RUNFILE} ]]; then
        source ${RUNFILE}
        rm ${RUNFILE}
        kill ${PID}
        echo "Waiting for Hikkabot instance death, PID: ${PID}"
        tail -f ${LOGDIR}/main.log | while read LOGLINE; do
            [[ "${LOGLINE}" == *"[main] Exit"* ]] && pkill -P $$ tail
        done
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
            sleep 5
            check $CONFIG $CHAT
#        else
#            STATS=`ps -p ${PID} -o %cpu,%mem | tail -1`
#            notify ${CONFIG} ${CHAT} "${STATS}"
        fi
    else
        notify ${CONFIG} ${CHAT} "Runfile not found." 1
    fi
}

update() {
    CONFIG=$1
    stop
    go get -u ${PACKAGE}
    go install ${PACKAGE}
    start ${CONFIG}
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

    "update")
        update $2
        ;;
esac