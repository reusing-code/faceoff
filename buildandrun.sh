#!/bin/bash

killall webserver

./buildPackage.sh
cd package
./webserver &