// Copyright (c) The Cryptorium Authors.
// Licensed under the MIT License.

package main

import (
	"log"
	stdlog "log"
	"path/filepath"
	"reflect"

	"github.com/alecthomas/kong"
	"github.com/cryptoriums/contraget/pkg/contraget"
	"github.com/nanmu42/etherscan-api"
	"github.com/pkg/errors"
)

var cli struct {
	Addr        string            `required:"" type:"string" help:"the contract address"`
	Network     etherscan.Network `default:"rinkeby" help:"the contract address"`
	Name        string            `required:"" type:"string" help:"the cli.Name for the downloaded contract"`
	DownloadDst string            `optional:"" type:"string" help:"the destination folder for the downloaded contract"`
	AbiDst      string            `optional:"" type:"string" help:"the destination folder for the abi generation"`
	PkgDst      string            `optional:"" type:"string" help:"the destination folder for the golang binding package"`
	PkgAliases  map[string]string `optional:"" type:"string:string" help:"alias for pgk bindings to use when there is a collision in the normalized names"`
}

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
		default:
			return errors.Errorf("unrecognized network cli.Name:%v", value)
		}
		return nil
	}
}

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile | log.Lmsgprefix)

	_ = kong.Parse(&cli, kong.UsageOnError(), kong.TypeMapper(reflect.TypeOf(etherscan.Network("")), networkDecoder()))

	downloadFolder := filepath.Join(cli.DownloadDst, cli.Name)

	filePaths, err := contraget.DownloadContracts(cli.Network, cli.Addr, downloadFolder, cli.Name)
	ExitOnErr(err, "download contracts")
	log.Printf("Downloaded contract:%+v", downloadFolder)

	if cli.PkgDst != "" {
		types, abis, bins, sigs, libs, err := contraget.GetContractObjects(filePaths)
		ExitOnErr(err, "get contracts object")

		err = contraget.GeneratePackage(cli.PkgDst, cli.Name, types, abis, bins, sigs, libs, cli.PkgAliases)
		ExitOnErr(err, "generate GO binding")

		log.Println("generated GO binding:", filepath.Join(cli.PkgDst, cli.Name))
	}

	if cli.AbiDst != "" {
		_, abis, _, _, _, err := contraget.GetContractObjects(filePaths)
		ExitOnErr(err, "get contracts object")
		err = contraget.GenerateABI(cli.AbiDst, cli.Name, abis)
		ExitOnErr(err, "generate ABI")
		log.Println("Saved ABI:", filepath.Join(downloadFolder, cli.Name))
	}

}

func ExitOnErr(err error, msg string) {
	if err != nil {
		stdlog.Fatalf("root execution error:%+v msg:%+v", err, msg)
	}
}
