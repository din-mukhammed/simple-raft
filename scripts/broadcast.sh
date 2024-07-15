#!/bin/bash

curl -v -X POST -L http://localhost:8080/broadcast \
      -d "{\"msg\":\"$1\"}"


