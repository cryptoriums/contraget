module github.com/cryptoriums/contraget

go 1.16

require (
	github.com/alecthomas/kong v0.2.15
	github.com/ethereum/go-ethereum v1.10.7
	github.com/nanmu42/etherscan-api v1.1.1
	github.com/pkg/errors v0.9.1
	go.uber.org/multierr v1.6.0
)

replace github.com/nanmu42/etherscan-api => github.com/cryptoriums/etherscan-api v1.3.1-0.20210819094440-d2f683c2d35c
