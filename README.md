# goproxy-p2p

`goproxy-p2p` is an [GOPROXY service](https://golang.org/ref/mod#module-proxy) that enables local network discovery of cooperating instances.

The agent discovers other instances using mulitcast [DNS service discovery](http://www.dns-sd.org/). Agent-to-agent communication is tunneled via
the ssh protocol, utilizing keys from the user's `ssh-agent` for the service's host keys. Agents provide both a client & a server. Clients use
the identities available in the local `ssh-agent` to connect to services. Servers utilize the sum of `ssh-agent` identities &
the key fingerprints in users' `.ssh/authorized_keys` to generate a list of identities allowed to connect. In this manner, we establish
that cooperating agents share the same user identity. This is necessary because the cache may contain repositories containing private
sources.

The GOPROXY service is bound to the loopback interface (`127.0.0.1`). When a `go` client utilizes the proxy, the proxy fans out to cooperating
instances and serves the Go module from the first replying instance.

## FAQ

1. Is this an offline proxy?
No. It does not work offline.

## Security concerns

1. I don't want to leak git repos from my work PC to my home computer
   Run with `-serve=false`. This disables the caching server from exporting across the network, but enables the agent to connect to serving
   instances.

## Testing

Here's how to run it in dev; add to bashrc etc:
```sh
$(cd ~/Projects/goproxy-p2p && go run ./cmd/goproxy-p2p/... -port 8080 -password ok -eval true)
```

Here's how to test how fast this is
```sh
sudo rm -rf /tmp/goscratch
GOPROXY=http://localhost:8080?pass=ok GOMODCACHE=/tmp/goscratch go get github.com/moby/moby
```


## Known Issues

1. Uses `SSH_AUTH_SOCK` when connecting via `ssh -A` (`AgentForwarding`).

   Probably want to add directions on running this with `launchctl` instead of forking on eval.
