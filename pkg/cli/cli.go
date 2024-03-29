// Copyright (c) The Cryptorium Authors.
// Licensed under the MIT License.

package cli

import (
	"log"
	"os"

	"github.com/cryptoriums/packages/compiler"
	etherscan_p "github.com/cryptoriums/packages/etherscan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/nanmu42/etherscan-api"
	"github.com/pkg/errors"
)

type Cli struct {
	Path            string            `required:"" type:"string" help:"the contract address or local file path"`
	CompilerVersion string            `type:"string" help:"the contract compiler version"`
	Network         etherscan.Network `default:"rinkeby" help:"the network to connect to"`
	DownloadDst     string            `optional:"" default:"/tmp" type:"string" help:"the destination folder for the downloaded contract"`
	ObjectsDst      string            `optional:"" type:"string" help:"the destination folder for the all object generation"`
	PkgDst          string            `optional:"" type:"string" help:"the destination folder for the golang binding package"`
	PkgAliases      map[string]string `optional:"" type:"string:string" help:"alias for pgk bindings to use when there is a collision in the normalized names"`
	CompilerArgs    []string          `optional:"" sep:";" help:"custom compiler arguments"`
}

func Run(cli *Cli) error {
	var filePaths map[string]string
	_, err := os.Stat(cli.Path)
	if err != nil {
		log.Printf("path not found localy so trying from etherscan:%v", cli.Path)
		if !common.IsHexAddress(cli.Path) {
			return errors.New("contract path is not a hex string")
		}

		filePaths, err = etherscan_p.DownloadContracts(cli.Network, cli.Path, cli.DownloadDst)
		if err != nil {
			return errors.Wrap(err, "download contracts")
		}
		log.Printf("Downloaded contract:%v to:%+v", cli.Path, cli.DownloadDst)
	} else {
		compilerVer := cli.CompilerVersion
		if compilerVer == "" {
			compilerVer, err = compiler.CompilerVersion(cli.Path)
			if err != nil {
				return errors.Wrap(err, "get contracts compiler version")
			}
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
		types, abis, bins, sigs, libs, err := compiler.GetContractObjects(filePaths, cli.CompilerArgs)
		if err != nil {
			return errors.Wrap(err, "get contracts object")
		}

		err = compiler.ExportPackage(cli.PkgDst, types, abis, bins, sigs, libs, cli.PkgAliases)
		if err != nil {
			return errors.Wrap(err, "generate GO binding")
		}
		log.Println("generated GO binding:", cli.PkgDst)
	} else {
		log.Println("no package destination set so skipping binding generation")
	}

	if cli.ObjectsDst != "" {
		types, abis, bins, _, _, err := compiler.GetContractObjects(filePaths, cli.CompilerArgs)
		if err != nil {
			return errors.Wrap(err, "get contracts object")
		}
		err = compiler.ExportABI(cli.ObjectsDst, abis)
		if err != nil {
			return errors.Wrap(err, "Export ABI")
		}
		log.Println("Exportd ABI:", cli.ObjectsDst)

		err = compiler.ExportBin(cli.ObjectsDst, types, bins)
		if err != nil {
			return errors.Wrap(err, "Export Bins")
		}
		log.Println("Exportd BINS:", cli.ObjectsDst)

	}
	return nil
}
