// Copyright (c) The Cryptorium Authors.
// Licensed under the MIT License.

package contraget

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/nanmu42/etherscan-api"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

func DownloadContracts(network etherscan.Network, address string, dstFolder, name string) (map[string]string, error) {
	client := etherscan.New(network, "")
	rep, err := client.ContractSource(address)
	if err != nil {
		return nil, errors.Wrap(err, "get contract source")
	}

	name = strings.Title(name)
	dstPath := path.Join(dstFolder, name)

	if _, err := os.Stat(dstPath); !os.IsNotExist(err) {
		os.RemoveAll(dstPath)
	}
	if err := os.MkdirAll(dstFolder, os.ModePerm); err != nil {
		return nil, errors.Wrapf(err, "create download folder:%v", dstFolder)
	}

	var contractFiles = make(map[string]string)

	if codes, ok := isMultiContract(rep[0].SourceCode); ok {
		for filePath := range codes {
			content := codes[filePath].Content
			filePath := filepath.Join(dstFolder, filepath.Base(filePath))
			if err := write(filePath, content); err != nil {
				return nil, err
			}
			contractFiles[filePath] = strings.Split(rep[0].CompilerVersion, "+")[0]
		}
	} else {
		if strings.Contains(rep[0].CompilerVersion, "vyper") {
			filePath := filepath.Join(dstFolder, name+".vy")
			if err := write(filePath, rep[0].SourceCode); err != nil {
				return nil, err
			}
			contractFiles[filePath] = "v" + strings.Split(rep[0].CompilerVersion, ":")[1]
		} else {
			filePath := filepath.Join(dstFolder, name+".sol")
			if err := write(filePath, rep[0].SourceCode); err != nil {
				return nil, err
			}
			contractFiles[filePath] = strings.Split(rep[0].CompilerVersion, "+")[0]
		}
	}

	return contractFiles, nil
}

func write(filePath, content string) (errFinal error) {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			errFinal = multierr.Append(errFinal, err)
		}
	}()
	w := bufio.NewWriter(f)
	defer func() {
		if err := w.Flush(); err != nil {
			errFinal = multierr.Append(errFinal, err)
		}
	}()

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		// Flatten the import subtree.
		// Rewrite the imports to remove all parent folder.
		if strings.HasPrefix(line, "import") {
			last := strings.LastIndex(line, "/")
			line = "import \"./" + line[last+1:len(line)-2] + "\";"
		}
		line += "\n"
		if _, err := w.Write([]byte(line)); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func GetContractObjects(contractFiles map[string]string) (types []string, abis []string, bins []string, sigs []map[string]string, libs map[string]string, err error) {
	libs = make(map[string]string)
	for contractPath, compilerVersion := range contractFiles {
		var contracts map[string]*compiler.Contract
		if filepath.Ext(contractPath) == ".sol" {
			compilerPath, err := downloadSolc(compilerVersion)
			if err != nil {
				return nil, nil, nil, nil, nil, errors.Wrap(err, "download solc")
			}
			contracts, err = compiler.CompileSolidity(compilerPath, contractPath)
			if err != nil {
				return nil, nil, nil, nil, nil, errors.Wrap(err, "build Solidity contract")
			}
		} else {
			compilerPath, err := downloadVyper(compilerVersion)
			if err != nil {
				return nil, nil, nil, nil, nil, errors.Wrap(err, "download solc")
			}
			output, err := compiler.CompileVyper(compilerPath, contractPath)
			if err != nil {
				return nil, nil, nil, nil, nil, errors.Wrap(err, "build Vyper contract")
			}
			contracts = make(map[string]*compiler.Contract)
			for n, contract := range output {
				name := n
				// Sanitize the combined json names to match the
				// format expected by solidity.
				if !strings.Contains(n, ":") {
					// Remove extra path components
					name = abi.ToCamelCase(strings.TrimSuffix(filepath.Base(name), ".vy"))
				}
				contracts[name] = contract
			}
		}

		for name, contract := range contracts {
			abi, err := json.Marshal(contract.Info.AbiDefinition)
			if err != nil {
				return nil, nil, nil, nil, nil, errors.Wrap(err, "flatten the compiler parse")
			}
			abis = append(abis, string(abi))
			bins = append(bins, contract.Code)
			sigs = append(sigs, contract.Hashes)
			nameParts := strings.Split(name, ":")
			types = append(types, nameParts[len(nameParts)-1])

			libPattern := crypto.Keccak256Hash([]byte(name)).String()[2:36]
			libs[libPattern] = nameParts[len(nameParts)-1]
		}

	}

	return types, abis, bins, sigs, libs, nil
}

func ExportABI(folder, filename string, abis []string) error {
	var a []byte
	for _, abi := range abis {
		if len(abi) > 2 {
			a = append(a, abi[1:len(abi)-1]...)
			a = append(a, []byte(",")...)

		}
	}
	a = a[:len(a)-1] // Remove the last comma from the array.
	a = append([]byte(`[`), a...)
	a = append(a, []byte("]")...)

	if err := os.MkdirAll(folder, os.ModePerm); err != nil {
		return errors.Wrapf(err, "create destination folder:%v", folder)
	}

	fpath := filepath.Join(folder, filename+".json")
	if err := ioutil.WriteFile(fpath, a, os.ModePerm); err != nil {
		return errors.Wrapf(err, "write file:%v", fpath)
	}

	return nil
}

func ExportBin(folder string, types, bins []string) error {
	for i, t := range types {
		fpath := filepath.Join(folder, t+".bin")
		if err := ioutil.WriteFile(fpath, []byte(bins[i]), os.ModePerm); err != nil {
			return errors.Wrapf(err, "write file:%v", fpath)
		}
	}
	return nil
}

func ExportPackage(pkgFolder, pkgName string, types []string, abis []string, bins []string, sigs []map[string]string, libs map[string]string, aliases map[string]string) error {
	code, err := bind.Bind(types, abis, bins, sigs, pkgName, bind.LangGo, libs, aliases)
	if err != nil {
		return errors.Wrapf(err, "generate the Go wrapper:%v", pkgName)
	}
	pkgFolderName := filepath.Join(pkgFolder, pkgName)

	pkgPath := filepath.Join(pkgFolderName, pkgName+".go")

	if _, err := os.Stat(pkgFolderName); !os.IsNotExist(err) {
		os.RemoveAll(pkgFolderName)
	}
	if err := os.MkdirAll(pkgFolderName, os.ModePerm); err != nil {
		return errors.Wrapf(err, "create destination folder:%v", pkgFolderName)
	}

	if err := ioutil.WriteFile(pkgPath, []byte(code), os.ModePerm); err != nil {
		return errors.Wrapf(err, "write package file:%v", pkgPath)
	}
	return nil
}

// downloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func downloadFile(filepath string, url string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "gettings the file")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("downloading solc returned unexpected status code:%v", resp.StatusCode)
	}

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return errors.Wrap(err, "creating destination file")
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return errors.Wrap(err, "writing the file")
}

// downloadSolc will download @solcVersion of the Solc compiler to tmp/solc directory.
func downloadSolc(version string) (string, error) {
	solcDir := filepath.Join("tmp", "solc")
	if err := os.MkdirAll(solcDir, os.ModePerm); err != nil {
		return "", err
	}
	solcPath := filepath.Join(solcDir, version)
	if _, err := os.Stat(solcPath); os.IsNotExist(err) {
		log.Println("downloading solc version", version)

		srcFile := ""
		switch runtime.GOOS {
		case "darwin":
			srcFile = "solc-macos"
		case "linux":
			srcFile = "solc-static-linux"
		default:
			return "", errors.Errorf("unsuported OS:%v", runtime.GOOS)
		}

		err = downloadFile(solcPath, fmt.Sprintf("https://github.com/ethereum/solidity/releases/download/%s/%s", version, srcFile))
		if err != nil {
			return "", err
		}
		if err := os.Chmod(solcPath, os.ModePerm); err != nil {
			return "", err
		}
	}
	return solcPath, nil
}

// downloadVyper will download the downloadVyper.
func downloadVyper(version string) (string, error) {
	compilerDir := filepath.Join("tmp", "vyper")
	if err := os.MkdirAll(compilerDir, os.ModePerm); err != nil {
		return "", err
	}
	vyperPath := filepath.Join(compilerDir, version)
	if _, err := os.Stat(vyperPath); os.IsNotExist(err) {
		log.Println("downloading vyper version", version)

		srcFile := ""
		switch version {
		case "v0.2.5":
			srcFile = "0.2.5+commit.a0c561c"
		case "v0.2.4":
			srcFile = "0.2.4+commit.7949850"
		default:
			return "", errors.Errorf("unrecognized version:%v", version)
		}
		switch runtime.GOOS {
		case "darwin":
			srcFile = "vyper." + srcFile + ".darwin"
		case "linux":
			srcFile = "vyper." + srcFile + ".linux"
		default:
			return "", errors.Errorf("unsuported OS:%v", runtime.GOOS)
		}

		err = downloadFile(vyperPath, fmt.Sprintf("https://github.com/vyperlang/vyper/releases/download/%s/%s", version, srcFile))
		if err != nil {
			return "", err
		}
		if err := os.Chmod(vyperPath, os.ModePerm); err != nil {
			return "", err
		}
	}
	return vyperPath, nil
}

type MultiContract struct {
	Language string
	Sources  map[string]Src
}

type Src struct {
	Content string
}

func isMultiContract(s string) (map[string]Src, bool) {
	out := &MultiContract{}

	// Etherscan has inconsistent api responses so need to deal with these here.
	if err := json.Unmarshal([]byte(s), &out.Sources); err == nil {
		return out.Sources, true
	}

	s = strings.ReplaceAll(s, "{{", "{") // Deal with another wierdness of etherscan.
	s = strings.ReplaceAll(s, "}}", "}")

	if err := json.Unmarshal([]byte(s), out); err == nil {
		return out.Sources, true
	}
	return nil, false
}
