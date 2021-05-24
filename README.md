# 

## FAQ

1. Is this an offline proxy?
No. It does not work offline.


## Testing
Here's how to test how fast this is
```
sudo rm -rf /tmp/goscratch
GOPROXY=http://localhost:8080 GOMODCACHE=/tmp/goscratch go get github.com/moby/moby
```