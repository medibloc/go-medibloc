// Copyright (C) 2018  MediBloc
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/medibloc/go-medibloc/keystore"

	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	imported := false
	var privKey string
	if len(os.Args) == 1 {
		privKey = ""

	} else if len(os.Args) == 3 && os.Args[1] == "-i" {
		pkString, err := ioutil.ReadFile(os.Args[2])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		privKey = string(pkString)
		imported = true
	} else {
		fmt.Println("usage: keyutil\n       keyutil [-i] key_file")
		os.Exit(1)
	}

	fmt.Print("Set your passphrase: ")
	passphrase, err := terminal.ReadPassword(0)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println()
	fmt.Print("Confirm your passphrase: ")
	passphrase2, err := terminal.ReadPassword(0)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println()
	if string(passphrase) != string(passphrase2) {
		fmt.Println("Confirmed passphrase doesn't match, please try again.")
		os.Exit(1)
	}
	err = keystore.MakeKeystoreV3(privKey, string(passphrase), "cmd/keyutil/"+time.Now().Format(time.RFC850)+" private.key")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if imported {
		fmt.Println("Keystore file successfully imported.")
	} else {
		fmt.Println("Keystore file successfully created.")
	}
}
