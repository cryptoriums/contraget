# Blockchain contract downloader
[![CI Checks](https://github.com/cryptoriums/contraget/workflows/Checks/badge.svg)](https://github.com/cryptoriums/contraget/actions?query=workflow%3Achecks)
[![Go Report Card](https://goreportcard.com/badge/github.com/cryptoriums/contraget)](https://goreportcard.com/report/github.com/cryptoriums/contraget)

## Main features
 - Download a verified contract from etherscan
 - Generate contract abi in json
 - Generate golang bindings

## Example
```
go run cmd/contraget/contraget.go --addr=0x34319564f00C924dA8fB52fD8bA6edBfd1FfEdA8 --download-dst=tmp --pkg-dst=pkg/contracts --network=goerli --name=tellorTest
```

## Initial Author
[@krasi-georgiev](https://github.com/krasi-georgiev/) as part of the Tellor team.
