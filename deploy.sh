#!/bin/bash

TMP=/tmp/hikkabot-deploy
BIN=hikkabot
WD=/home/ryan/apps/hikkabot
GOOS=linux GOARCH=amd64 go build -o bin/$BIN
ssh root@vps "rm -r $TMP; mkdir $TMP"
scp bin/hikkabot root@vps:/tmp/hikkabot-deploy
ssh root@vps "supervisorctl stop hikkabot; mv $TMP/$BIN $WD; supervisorctl start hikkabot"