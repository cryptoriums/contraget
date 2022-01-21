// Copyright (c) The Cryptorium Authors.
// Licensed under the MIT License.

package main

import (
	"log"
	stdlog "log"
	"os"
	"path/filepath"
	"reflect"

	"github.com/alecthomas/kong"
	"github.com/cryptoriums/contraget/pkg/contraget"
	"github.com/cryptoriums/packages/ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/nanmu42/etherscan-api"
	"github.com/pkg/errors"
)

type cli struct {
	Path        string            `required:"" type:"string" help:"the contract address or local file path"`
	SolcVersion string            `type:"string" help:"the contract compiler version"`
	Network     etherscan.Network `default:"rinkeby" help:"the network to connect to"`
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

	cli := &cli{}

	_ = kong.Parse(cli, kong.UsageOnError(), kong.TypeMapper(reflect.TypeOf(etherscan.Network("")), networkDecoder()))

	var filePaths map[string]string
	_, err := os.Stat(cli.Path)
	if err != nil {
		if !common.IsHexAddress(cli.Path) {
			cli.ExitOnErr(errors.New("contract path is not a hex string"), "")
		}
		downloadFolder := filepath.Join(cli.DownloadDst, cli.Name)

		filePaths, err = contraget.DownloadContracts(cli.Network, cli.Path, downloadFolder, cli.Name)
		cli.ExitOnErr(err, "download contracts")
		log.Printf("Downloaded contract:%+v", downloadFolder)
	} else {
		compilerVer := cli.SolcVersion
		if compilerVer == "" {
			compilerVer, err = ethereum.CompilerVersion(cli.Path)
			cli.ExitOnErr(err, "get contracts compiler version")
			log.Printf("compiler version not specified so inferred from the contract version:%v", compilerVer)
		}

		if compilerVer[0:1] != "v" {
			compilerVer = "v" + compilerVer
		}
		filePaths = map[string]string{
			cli.Path: compilerVer,
		}
	}

	if cli.PkgDst != "" {
		types, abis, bins, sigs, libs, err := contraget.GetContractObjects(filePaths)
		cli.ExitOnErr(err, "get contracts object")

		err = contraget.GeneratePackage(cli.PkgDst, cli.Name, types, abis, bins, sigs, libs, cli.PkgAliases)
		cli.ExitOnErr(err, "generate GO binding")

		log.Println("generated GO binding:", filepath.Join(cli.PkgDst, cli.Name))
	} else {
		log.Println("no package destination set so skipping binding generation")
	}

	if cli.AbiDst != "" {
		_, abis, _, _, _, err := contraget.GetContractObjects(filePaths)
		cli.ExitOnErr(err, "get contracts object")
		err = contraget.GenerateABI(cli.AbiDst, cli.Name, abis)
		cli.ExitOnErr(err, "generate ABI")
		log.Println("Saved ABI:", filepath.Join(cli.AbiDst, cli.Name))
	}

}

func (self *cli) ExitOnErr(err error, msg string) {
	if err != nil {
		stdlog.Fatalf("root execution name:%v, error:%+v msg:%+v", self.Name, err, msg)
	}
}
