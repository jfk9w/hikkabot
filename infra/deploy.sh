#!/bin/bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

GOOS=linux GOARCH=amd64 go build -o $DIR/../bin/hikkabot
if [ $? -ne 0 ]; then
    exit 1
fi

ansible-playbook "$DIR/deploy.yml"