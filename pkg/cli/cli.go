package cli

import (
	"log"
	"os"
	"path/filepath"

	"github.com/cryptoriums/contraget/pkg/contraget"
	"github.com/cryptoriums/packages/ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/nanmu42/etherscan-api"
	"github.com/pkg/errors"
)

type Cli struct {
	Path        string            `required:"" type:"string" help:"the contract address or local file path"`
	SolcVersion string            `type:"string" help:"the contract compiler version"`
	Network     etherscan.Network `default:"rinkeby" help:"the network to connect to"`
	Name        string            `required:"" type:"string" help:"the name for the downloaded contract"`
	DownloadDst string            `optional:"" type:"string" help:"the destination folder for the downloaded contract"`
	ObjectsDst  string            `optional:"" type:"string" help:"the destination folder for the all object generation"`
	PkgDst      string            `optional:"" type:"string" help:"the destination folder for the golang binding package"`
	PkgAliases  map[string]string `optional:"" type:"string:string" help:"alias for pgk bindings to use when there is a collision in the normalized names"`
}

func Run(cli *Cli) error {
	var filePaths map[string]string
	_, err := os.Stat(cli.Path)
	if err != nil {
		log.Printf("path not found so trying from etherscan:%v", cli.Path)
		if !common.IsHexAddress(cli.Path) {
			return errors.New("contract path is not a hex string")
		}
		downloadFolder := filepath.Join(cli.DownloadDst, cli.Name)

		filePaths, err = contraget.DownloadContracts(cli.Network, cli.Path, downloadFolder, cli.Name)
		if err != nil {
			return errors.Wrap(err, "download contracts")
		}
		log.Printf("Downloaded contract:%+v", downloadFolder)
	} else {
		compilerVer := cli.SolcVersion
		if compilerVer == "" {
			compilerVer, err = ethereum.CompilerVersion(cli.Path)
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
		types, abis, bins, sigs, libs, err := contraget.GetContractObjects(filePaths)
		if err != nil {
			return errors.Wrap(err, "get contracts object")
		}

		err = contraget.ExportPackage(cli.PkgDst, cli.Name, types, abis, bins, sigs, libs, cli.PkgAliases)
		if err != nil {
			return errors.Wrap(err, "generate GO binding")
		}
		log.Println("generated GO binding:", filepath.Join(cli.PkgDst, cli.Name))
	} else {
		log.Println("no package destination set so skipping binding generation")
	}

	if cli.ObjectsDst != "" {
		types, abis, bins, _, _, err := contraget.GetContractObjects(filePaths)
		if err != nil {
			return errors.Wrap(err, "get contracts object")
		}
		err = contraget.ExportABI(cli.ObjectsDst, cli.Name, abis)
		if err != nil {
			return errors.Wrap(err, "Export ABI")
		}
		log.Println("Exportd ABI:", filepath.Join(cli.ObjectsDst, cli.Name))

		err = contraget.ExportBin(cli.ObjectsDst, types, bins)
		if err != nil {
			return errors.Wrap(err, "Export Bins")
		}
		log.Println("Exportd BINS:", filepath.Join(cli.ObjectsDst, cli.Name))

	}
	return nil
}
