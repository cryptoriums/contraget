// Copyright (c) The Cryptorium Authors.
// Licensed under the MIT License.

package main

import (
	"log"
	stdlog "log"
	"reflect"

	"github.com/alecthomas/kong"
	"github.com/cryptoriums/contraget/pkg/cli"
	"github.com/nanmu42/etherscan-api"
	"github.com/pkg/errors"
)

func networkDecoder() kong.MapperFunc {
	return func(ctx *kong.DecodeContext, target reflect.Value) error {
		var value string
		if err := ctx.Scan.PopValueInto("network", &value); err != nil {
			return err
		}
		switch value {
		case "rinkeby":
			target.Set(reflect.ValueOf(etherscan.Rinkby))
		case "mainnet":
			target.Set(reflect.ValueOf(etherscan.Mainnet))
		case "goerli":
			target.Set(reflect.ValueOf(etherscan.Goerli))
		default:
			return errors.Errorf("unrecognized network cli.Name:%v", value)
		}
		return nil
	}
}

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile | log.Lmsgprefix)

	cliI := &cli.Cli{}

	_ = kong.Parse(cliI, kong.UsageOnError(), kong.TypeMapper(reflect.TypeOf(etherscan.Network("")), networkDecoder()))

	if err := cli.Run(cliI); err != nil {
		stdlog.Fatalf("root execution path:%v, error:%+v", cliI.Path, err)
	}
}
