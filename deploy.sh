#!/bin/bash

cd ..
CWD=$(pwd)
REPOS="hikkabot jfk9w-go/telegram-bot-api jfk9w-go/flu"
for REPO in $REPOS; do
  cd $REPO
  go clean
  go mod tidy
  cd $CWD
done

tar -cvjf hikkabot.tar.bz2 --exclude='.git' --exclude='.idea' $REPOS
scp hikkabot.tar.bz2 pi@pi.local:/home/pi
ssh pi@pi.local "(rm -r /home/pi/src/hikkabot || true) && \
(rm -r /home/pi/src/jfk9w-go/telegram-bot-api) || true && \
(rm -r /home/pi/src/jfk9w-go/flu || true) && \
mkdir -p /home/pi/src && \
tar -C src -xvjf hikkabot.tar.bz2 && \
cd /home/pi/src/hikkabot && \
CGO_ENABLED=1 /usr/local/go/bin/go build -v . &&
mv hikkabot /home/pi/hikkabot/hikkabot"
ssh pi.local "sudo supervisorctl restart hikkabot"