module jonwillia.ms/goproxy-p2p

go 1.16

require (
	github.com/goproxy/goproxy v0.6.2
	github.com/grandcat/zeroconf v1.0.0
	github.com/miekg/dns v1.1.42 // indirect
	github.com/rs/zerolog v1.22.0
	github.com/sirupsen/logrus v1.8.1 // indirect
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	golang.org/x/net v0.0.0-20210521195947-fe42d452be8f // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210521203332-0cec03c779c1 // indirect
	jonwillia.ms/weyoun v0.0.0-00010101000000-000000000000
)

replace jonwillia.ms/weyoun => ../weyoun
