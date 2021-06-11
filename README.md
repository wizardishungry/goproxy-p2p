# 

## FAQ

1. Is this an offline proxy?
No. It does not work offline.


## Security concerns

1. I don't want to leak git repos from my work PC to my home computer
   Run with `-serve=false`

## Testing
Here's how to test how fast this is
```
sudo rm -rf /tmp/goscratch
GOPROXY=http://localhost:8080 GOMODCACHE=/tmp/goscratch go get github.com/moby/moby
```

Here's how to run it in dev- add to bashrc etc:
```
$(cd ~/Projects/goproxy-p2p && go run ./cmd/goproxy-p2p/... -port 8080 -password ok -eval true)
```