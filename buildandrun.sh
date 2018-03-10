#!/bin/bash

cd client
gopherjs build
cd ../webserver
killall webserver
go build
./webserver &