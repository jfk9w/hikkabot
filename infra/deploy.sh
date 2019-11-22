#!/bin/bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

BIN=$DIR/../bin
ARMLIB=arm-unknown-linux-gnueabi
ARMPATH=$BIN/$ARMLIB
if [ ! -d "$ARMPATH" ]; then
  DLPATH=$TMPDIR/arm.tar.xz
  echo "Downloading ARM libraries..."
  curl https://raw.githubusercontent.com/jfk9w-go/lib/master/darwin/arm-unknown-linux-gnueabi.tar.xz > $DLPATH
  echo "Unpacking ARM libraries to $ARMPATH"
  tar -xf $DLPATH -C $BIN
  if [ $? -ne 0 ]; then
    echo "Failed to unpack ARM libraries"
    exit 1
  fi

  rm $DLPATH
fi

export PATH=$PATH:$ARMPATH/bin
CC=$ARMLIB-gcc CXX=$ARMLIB-g++ CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=7 go build -o $DIR/../bin/hikkabot
if [ $? -ne 0 ]; then
    exit 1
fi

echo -n "sudo password: "
read -s SUDO_PWD

ansible-playbook "$DIR/deploy.yml" --extra-vars "ansible_sudo_pass=$SUDO_PWD"