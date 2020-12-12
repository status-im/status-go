#!/bin/bash
for ((i=1;i<=100000000000;i++))
do
        /Users/Franklyn/development/project/from_github/status-go/cmd/spam/spam -s 2 -t 1 -m "oi"
        rm -rf ./app-* Users data
        sleep 1s
done
