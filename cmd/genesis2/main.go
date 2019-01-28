package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

func printUsage() {
	fmt.Println("generate auto")
	fmt.Println("generate pre <output file>: generate pre configuration")
	fmt.Println("generate final <pre config file> <genesis output> <proposer output>: generate genesis configuration")
}

func main() {
	if len(os.Args) == 3 && os.Args[1] == "pre" {
		out := preConfig()
		err := ioutil.WriteFile(os.Args[2], out, 0600)
		if err != nil {
			panic(err)
		}
		return
	} else if len(os.Args) == 5 && os.Args[1] == "final" {
		buf, err := ioutil.ReadFile(os.Args[2])
		if err != nil {
			printUsage()
			os.Exit(1)
		}

		genesis, proposer := finalConfig(buf)
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
		pre := preConfig()
		genesis, proposer := finalConfig(pre)
		err := ioutil.WriteFile("genesis.conf", genesis, 0600)
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
