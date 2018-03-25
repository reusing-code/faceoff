#!/bin/bash

cd client
gopherjs build
cd ../webserver
go build
cd ..

mkdir -p package

cp -rf templates package/
cp -rf static package/
cp webserver/webserver package/
cp client/client.js package/static/
cp client/client.js.map package/static/
cp version.txt package/ 2>/dev/null || true
