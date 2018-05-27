#!/bin/bash

PATH=$PATH:$HOME/Go/bin

CONFIG=$HOME/.config/hikkabot.json
RUNFILE=$HOME/.hikkabot
LOGDIR=$HOME/hikkabot/logs
PACKAGE=github.com/jfk9w-go/hikkabot

archive_logs() {
    CWD=`pwd`
    DIR=`date +%Y-%m-%d_%H-%M-%S`
    cd ${LOGDIR}
    mkdir ${DIR}
    mv *.log ${DIR}
    cd ${CWD}
}

start() {
    if [[ -f ${RUNFILE} ]]; then
        echo "Hikkabot instance already running, PID: `cat ${RUNFILE}`"
    else
        mkdir -p ${LOGDIR}
        env CONFIG=${CONFIG} LOG=${CONFIG} hikkabot 2>&1 > ${LOGDIR}/main.log &
        echo -e "PID=$!" > ${RUNFILE}
        notify "RUNNING" 1
    fi
}

stop() {
    if [[ -f ${RUNFILE} ]]; then
        source ${RUNFILE}
        kill ${PID}
        if [[ $? -eq 0 ]]; then
            notify "SHUTDOWN" 1
            echo "Waiting for Hikkabot instance death, PID: ${PID}"
            tail -f ${LOGDIR}/main.log | while read LOGLINE; do
                [[ "${LOGLINE}" == *"[main] Exit"* ]] && pkill -P $$ tail
            done
        fi
        rm ${RUNFILE}
        archive_logs
        echo "OK"
    else
        echo "Hikkabot instance not running"
    fi
}

notify() {
    TEXT=`echo $1 | sed -r 's/\s+/%20/g;s/\./%2E/g'`
    NOTIFY=$2
    TOKEN=`cat ${CONFIG} | jq -r ".telegram | .token"`
    CHAT=`cat ${CONFIG} | jq -r ".mgmt"`
    FORM="chat_id=${CHAT}&text=${TEXT}"
    if [[ ! ${NOTIFY} ]]; then
        FORM="${FORM}&disable_notification=true"
    fi

    curl -s -d ${FORM} -X POST https://api.telegram.org/bot${TOKEN}/sendMessage > /dev/null
}

check() {
    if [[ -f ${RUNFILE} ]]; then
        source ${RUNFILE}
        kill -0 ${PID}
        if [[ $? -ne 0 ]]; then
            rm ${RUNFILE}
            notify "Instance is not running. Restarting." 1
            archive_logs
            start
            sleep 5
            check
#        else
#            STATS=`ps -p ${PID} -o %cpu,%mem | tail -1`
#            notify ${CONFIG} ${CHAT} "${STATS}"
        fi
    else
        notify "Runfile not found." 1
    fi
}

install() {
    go get -v -u ${PACKAGE}
    go install ${PACKAGE}
    cp ${GOPATH}/src/${PACKAGE}/bot.sh .
}

restart() {
    stop
    start
}

case $1 in
    "start")
        start
        ;;

    "stop")
        stop
        ;;

    "check")
        check
        ;;

    "install")
        install
        ;;

    "restart")
        restart
        ;;
esac