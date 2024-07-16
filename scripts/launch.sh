#!/bin/bash

LOG_LEVEL=trace \
SERVER_NAME=$1 \
APPLICATION_PORT=$2 \
SERVER_ID=$3 \
    ./raft $4
