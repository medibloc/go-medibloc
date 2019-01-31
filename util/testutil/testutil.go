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

package testutil

import (
	"strings"
	"testing"

	goNet "net"

	"time"

	"github.com/medibloc/go-medibloc/common/trie"
	"github.com/medibloc/go-medibloc/storage"
	"github.com/medibloc/go-medibloc/util/byteutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GetStorage return storage
func GetStorage(t *testing.T) storage.Storage {
	s, err := storage.NewMemoryStorage()
	assert.Nil(t, err)
	return s
}

// FindRandomListenPorts returns empty ports
func FindRandomListenPorts(n int) (ports []string) {
	listens := make([]goNet.Listener, 0)
	for i := 0; i < n; i++ {
		lis, _ := goNet.Listen("tcp", "127.0.0.1:0")
		addr := lis.Addr().String()
		ports = append(ports, strings.TrimLeft(addr, "127.0.0.1:"))
		listens = append(listens, lis)
	}
	for i := 0; i < n; i++ {
		listens[i].Close()
		for {
			conn, err := goNet.DialTimeout("tcp", listens[i].Addr().String(), time.Millisecond*50)
			if err != nil {
				break
			}
			conn.Close()
			time.Sleep(time.Millisecond * 50)
		}
	}

	return ports
}

// KeyOf find the key at which a given value can be found in the trie batch
func KeyOf(t *testing.T, trie *trie.Trie, value []byte) []byte {
	iter, err := trie.Iterator(nil)
	require.NoError(t, err)

	exist, err := iter.Next()
	for exist {
		require.NoError(t, err)
		if byteutils.Equal(iter.Value(), value) {
			return iter.Key()
		}
		exist, err = iter.Next()
	}
	return nil
}

// TrieLen counts the number of trie members
func TrieLen(t *testing.T, trie *trie.Batch) int {
	iter, err := trie.Iterator(nil)
	require.NoError(t, err)

	var cnt = 0
	exist, err := iter.Next()
	for exist {
		require.NoError(t, err)
		cnt++
		exist, err = iter.Next()
	}
	return cnt
}

// IP2Local changes ip address to localhost address
func IP2Local(ipAddr string) string {
	return "http://localhost:" + strings.Split(ipAddr, ":")[1]
}
