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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func main() {
	resp, err := http.Get("https://api.github.com/repos/btcsuite/btcd/commits?client_id=80386779008eea5dab41&client_secret=f2086aebf790729026fb209b803010029821d8a3")

	if err != nil {
		// handle error
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		// handle error
	}
	var f interface{}
	json.Unmarshal(body, &f)

	for _, n := range f.([]interface{}) {
		for k, v := range n.(map[string]interface{}) {
			if k == "commit" {
				value := v.(map[string]interface{})["message"].(string)
				fmt.Println(value)
				if strings.Contains(value, "gx publish ") {
					url := v.(map[string]interface{})["url"].(string)
					fmt.Println(v.(map[string]interface{})["message"])
					temps := strings.Split(url, "/")
					fmt.Println(temps[len(temps)-1])

					str := "github.com/agl/ed25519"
					temps = strings.Split(str, "ghub.com")
					fmt.Println(len(temps))
					fmt.Println(temps)
					return
				}
			}
		}
	}

}
