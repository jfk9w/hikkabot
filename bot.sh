#!/bin/bash

RED=31
GREEN=32

function color() {
    printf "\x1b[$2m$1\x1b[0m"
}

function mute() {
    $@ > /dev/null 2>&1
}

function prev() {
    if [[ $? -ne 0 ]]; then
        if [[ "$1" != "" ]]; then
            echo "$(color ERROR "$RED") $1"
        fi

        exit 1
    fi
}

function require() {
    local ok=1
    for cmd in "$@"; do
        mute type "$cmd"
        if [[ $? -ne 0 ]]; then
            ok=0
            echo "$cmd not found"
        fi
    done
    if [[ ok -ne 1 ]]; then
        exit 1
    fi
}

if [[ "$GOPATH" == "" ]]; then
    printf "GOPATH is not set"
    exit 1
fi

PATH="$PATH:$GOPATH/bin"
PACKAGE="github.com/jfk9w-go/hikkabot"

if [[ "$CONFIG" == "" ]]; then
    CONFIG="`pwd`/config.json"
fi

if [[ "$RUNFILE" == "" ]]; then
    PIDFILE="$HOME/.hikkabot"
fi

if [[ "$STDOUT" == "" ]]; then
    STDOUT="`pwd`/hikkabot.run"
fi

SED=sed
if [[ $(uname) == "Darwin" ]]; then
    SED=gsed
fi

function config() {
    cat "$CONFIG" | jq -r "$1"
}

function start() {
    if [[ -f "$PIDFILE" ]]; then
        mute . "$PIDFILE"
        if [[ $? -ne 0 || "$PID" == "" ]]; then
            echo "PID not set"
            exit 1
        fi

        echo "Hikkabot instance already running, PID: $PID"
        exit 2
    fi

    mkdir -p "$(dirname "$STDOUT")"
    CONFIG="$CONFIG" LOG="$CONFIG" hikkabot > "$STDOUT" 2>&1 &
    prev "Failed to start hikkabot"

    PID=$!
    echo -e "PID=$PID" > "$PIDFILE"
    echo "PID: $PID"
    notify "RUNNING" 1
}

function stop() {
    if [[ ! -f "$PIDFILE" ]]; then
        echo "Pidfile not found"
        exit 2
    fi

    mute . "$PIDFILE"
    if [[ $? -ne 0 || "$PID" == "" ]]; then
        echo "PID not set"
        exit 1
    else
        echo "PID: $PID"
    fi

    mute kill "$PID"
    if [[ $? -ne 0 ]]; then
        echo "Failed to kill process, cleanup"
        cleanup
        exit 2
    fi

    notify "SHUTDOWN" 1
    echo "Waiting for Hikkabot instance death, PID: $PID"
    tail -f "$STDOUT" | while read LOGLINE; do
        [[ "$LOGLINE" == *"[main] Exit"* ]] && pkill -P $$ tail
    done

    cleanup
    echo "OK"
}

function cleanup() {
    rm "$PIDFILE"
}

function notify() {
    TEXT=$(echo "$1" | "$SED" -r 's/\s+/%20/g;s/\./%2E/g')
    FORM="chat_id=$(config ".mgmt")&text=$TEXT"
    if [[ ! $2 ]]; then
        FORM="$FORM&disable_notification=true"
    fi

    mute curl -s -d "$FORM" -X POST "https://api.telegram.org/bot$(config ".telegram | .token")/sendMessage"
}

function check() {
    if [[ ! -f "$PIDFILE" ]]; then
        notify "Pidfile not found" 1
        exit 0
    fi

    mute . "$PIDFILE"
    if [[ $? -ne 0 || "$PID" == "" ]]; then
        echo "PID not set"
        exit 1
    else
        echo "PID: $PID"
    fi

    mute kill -0 "$PID"
    if [[ $? -ne 0 ]]; then
        cleanup
        notify "RESTART" 1
        start
        sleep 5
        check
    fi
}

function install() {
    go get -v -u "$PACKAGE"
    prev

    go install -v "$PACKAGE"
    prev

    local dir=.
    if [[ "$1" != "" ]]; then
        dir="$1"
    fi

    cp "$GOPATH/src/$PACKAGE/bot.sh" "$dir"
    prev
}

function restart() {
    stop
    if [[ $? -ne 0 ]]; then
        exit 1
    fi

    start
}

$@