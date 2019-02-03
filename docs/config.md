Websocket config nginx

```
location /ws/ {
                proxy_pass http://127.0.0.1:8086;
                proxy_http_version 1.1;
                proxy_set_header Upgrade $http_upgrade;
                proxy_set_header Connection "upgrade";
                proxy_read_timeout 2m;
                proxy_set_header Origin '';
        }
```
