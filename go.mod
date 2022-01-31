module github.com/cryptoriums/contraget

go 1.16

require (
	github.com/alecthomas/kong v0.2.15
	github.com/cryptoriums/packages v0.0.0-20220131140531-35c105c9cf21
	github.com/ethereum/go-ethereum v1.10.15
	github.com/nanmu42/etherscan-api v1.6.0
	github.com/pkg/errors v0.9.1
	go.uber.org/multierr v1.6.0
)

replace github.com/cryptoriums/packages => ../packages
