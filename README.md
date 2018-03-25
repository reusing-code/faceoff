# faceoff

faceoff is a small web app to decide *what is better* through an elemination tournament in one-against-one decision votings.

A live demo can be seen at [whatisbetter.de](https://whatisbetter.de) (currently only in german, sorry)

This is more than anything else a little project to get hands on experience with different technologies/concepts:
* [go](https://golang.org/) on server side
* [go](https://golang.org/) on client side (through [gopherjs](https://github.com/gopherjs/gopherjs))
* Single Page Applications (SPA)
    * Client side routing
    * HTML5 history API
    * Client side data storage/processing
    * Client side template rendering
* Websockets
* Simple Key/Value DB ([bbolt](https://github.com/coreos/bbolt)) instead of full SQL DBMS
* nginx as a reverse proxy with TLS (not part of this repository)
