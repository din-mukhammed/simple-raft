#!/bin/bash

curl -v -X POST http://localhost:8080/broadcast \
      -d "{\"msg\":\"$1\"}"


