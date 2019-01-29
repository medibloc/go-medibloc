package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/medibloc/go-medibloc/util/genesisutil"
)

func printUsage() {
	fmt.Println("generate auto")
	fmt.Println("generate pre <output file>: generate pre configuration")
	fmt.Println("generate final <pre config file> <genesis output> <proposer output>: generate genesis configuration")
}

func main() {
	if len(os.Args) == 3 && os.Args[1] == "pre" {
		conf := genesisutil.DefaultConfig()
		out, err := genesisutil.ConfigToBytes(conf)
		if err != nil {
			panic(err)
		}
		err = ioutil.WriteFile(os.Args[2], out, 0600)
		if err != nil {
			panic(err)
		}
		return
	} else if len(os.Args) == 5 && os.Args[1] == "final" {
		in, err := ioutil.ReadFile(os.Args[2])
		if err != nil {
			printUsage()
			os.Exit(1)
		}
		conf, err := genesisutil.BytesToConfig(in)
		if err != nil {
			panic(err)
		}
		genesis, err := genesisutil.ConvertGenesisConfBytes(conf)
		if err != nil {
			panic(err)
		}
		proposer := genesisutil.ConvertProposerConfBytes(conf)
		err = ioutil.WriteFile(os.Args[3], genesis, 0600)
		if err != nil {
			panic(err)
		}
		err = ioutil.WriteFile(os.Args[4], proposer, 0600)
		if err != nil {
			panic(err)
		}
		return
	} else if len(os.Args) == 2 && os.Args[1] == "auto" {
		conf := genesisutil.DefaultConfig()
		genesis, err := genesisutil.ConvertGenesisConfBytes(conf)
		if err != nil {
			panic(err)
		}
		proposer := genesisutil.ConvertProposerConfBytes(conf)
		err = ioutil.WriteFile("genesis.conf", genesis, 0600)
		if err != nil {
			panic(err)
		}
		err = ioutil.WriteFile("proposer.conf", proposer, 0600)
		if err != nil {
			panic(err)
		}

		return
	}
	printUsage()
	os.Exit(1)
}
