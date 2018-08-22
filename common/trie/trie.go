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

package trie

import (
	"errors"

	"bytes"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common/trie/pb"
	"github.com/medibloc/go-medibloc/storage"
	"golang.org/x/crypto/sha3"
)

// Errors
var (
	ErrInvalidNodeType = errors.New("invalid node type")
	ErrPbMsg           = errors.New("Pb Message cannot be converted into Node")
	ErrNotFound        = storage.ErrKeyNotFound
)

// Flag to identify the type of node
type ty uint8

// const Node types
const (
	unknown ty = iota
	branch
	ext
	leaf
)

// Node in trie, three kinds,
// Branch Node [hash_0, hash_1, ..., hash_f]
// Extension Node [flag, encodedPath, next hash]
// Leaf Node [flag, encodedPath, value]
type node struct {
	Type ty
	// Val branch: [17 children], ext: [path, next], leaf: [path, value]
	Val [][]byte
}

func (n *node) toProto() (proto.Message, error) {
	return &triepb.Node{
		Type: uint32(n.Type),
		Val:  n.Val,
	}, nil
}

func (n *node) fromProto(msg proto.Message) error {
	if msg, ok := msg.(*triepb.Node); ok {
		n.Type = ty(msg.Type)
		n.Val = msg.Val
		return nil
	}
	return ErrPbMsg
}

// Trie is a Merkle Patricia Trie, consists of three kinds of nodes,
// Branch Node: 16-elements array, value is [hash_0, hash_1, ..., hash_f, hash]
// Extension Node: 3-elements array, value is [ext flag, prefix path, next hash]
// Leaf Node: 3-elements array, value is [leaf flag, suffix path, value]
type Trie struct {
	rootHash []byte
	storage  storage.Storage
	// TODO add key length
}

// NewTrie create and return new Trie instance
func NewTrie(rootHash []byte, storage storage.Storage) (*Trie, error) {
	switch {
	case rootHash != nil:
		if _, err := storage.Get(rootHash); err != nil {
			return nil, err
		}
	}
	return &Trie{rootHash: rootHash, storage: storage}, nil
}

// Clone clone trie
func (t *Trie) Clone() (*Trie, error) {
	return &Trie{rootHash: t.rootHash, storage: t.storage}, nil
}

// Delete delete value from Trie
func (t *Trie) Delete(key []byte) error {
	hash, err := t.delete(t.rootHash, KeyToRoute(key))
	if err != nil {
		return err
	}
	t.rootHash = hash
	return nil
}

// Get get value from Trie
func (t *Trie) Get(key []byte) ([]byte, error) {
	return t.get(t.rootHash, KeyToRoute(key))
}

// Put put value to Trie
func (t *Trie) Put(key []byte, value []byte) error {
	hash, err := t.update(t.rootHash, KeyToRoute(key), value)
	if err != nil {
		return err
	}

	t.rootHash = hash
	return nil
}

// RootHash getter for rootHash
func (t *Trie) RootHash() []byte {
	return t.rootHash
}

// SetRootHash setter for rootHash
func (t *Trie) SetRootHash(rootHash []byte) {
	t.rootHash = rootHash
}

func (t *Trie) addExtToBranch(branchHash []byte, path []byte) ([]byte, error) {
	if path != nil && len(path) > 0 {
		var hash []byte
		var err error
		ext := newExtNode(path, branchHash)
		if hash, err = t.saveNode(ext); err != nil {
			return nil, err
		}
		return hash, nil
	}
	return branchHash, nil
}

func (t *Trie) delete(rootHash []byte, route []byte) ([]byte, error) {
	n, err := t.fetchNode(rootHash)
	if err != nil {
		return nil, err
	}
	var hash []byte
	switch n.Type {
	case branch:
		if len(route) == 0 {
			n.Val[16] = nil
			if hash, err = t.saveNode(n); err != nil {
				return nil, err
			}
		} else {
			if hash, err = t.delete(n.Val[route[0]], route[1:]); err != nil {
				return nil, err
			}
			n.Val[route[0]] = hash
		}
		var (
			childrenCnt = 0
			childIndex  = 0
		)

		for i := 0; i < 17; i++ {
			if n.Val[i] != nil && len(n.Val[i]) != 0 {
				childIndex = i
				childrenCnt++
			}
		}

		// branch has only one child node
		if childrenCnt < 2 {
			childNode, err := t.fetchNode(n.Val[childIndex])
			if err != nil {
				return nil, err
			}
			switch childNode.Type {
			case branch: // change branch to ext
				newExt := newExtNode([]byte{byte(childIndex)}, n.Val[childIndex])
				return t.saveNode(newExt)
			case ext: // compress branch to ext
				childNode.Val[0] = append([]byte{byte(childIndex)}, childNode.Val[0]...)
				return t.saveNode(childNode)
			case leaf: // compress branch to leaf
				if childIndex == 16 {
					return n.Val[16], nil
				}
				childNode.Val[0] = append([]byte{byte(childIndex)}, childNode.Val[0]...)
				return t.saveNode(childNode)
			}
		}

		return t.saveNode(n)
	case ext:
		path := n.Val[0]
		next := n.Val[1]
		matchLen := prefixLen(path, route)
		if matchLen != len(path) {
			return nil, ErrNotFound
		}
		if hash, err = t.delete(next, route[matchLen:]); err != nil {
			return nil, err
		}
		childNode, err := t.fetchNode(hash)
		if err != nil {
			return nil, err
		}
		switch childNode.Type {
		case branch: // change next hash to new branch hash
			n.Val[1] = hash
			return t.saveNode(n)
		case ext: // compress ext + ext to ext
			childNode.Val[0] = append(n.Val[0], childNode.Val[0]...)
			return t.saveNode(childNode)
		case leaf: // compress ext + leaf to leaf
			childNode.Val[0] = append(n.Val[0], childNode.Val[0]...)
			return t.saveNode(childNode)
		}
	case leaf:
		return nil, nil
	}
	return nil, ErrInvalidNodeType
}

func (t *Trie) fetchNode(hash []byte) (*node, error) {
	value, err := t.storage.Get(hash)
	if err != nil {
		return nil, err
	}
	pb := new(triepb.Node)
	if err := proto.Unmarshal(value, pb); err != nil {
		return nil, err
	}
	n := new(node)
	if err := n.fromProto(pb); err != nil {
		return nil, err
	}
	if n.Type < branch || n.Type > leaf {
		return nil, ErrInvalidNodeType
	}
	return n, nil
}

func (t *Trie) get(root []byte, route []byte) ([]byte, error) {
	curRootHash := root
	curRoute := route
	for len(curRoute) >= 0 {
		rootNode, err := t.fetchNode(curRootHash)
		if err != nil {
			return nil, err
		}
		switch rootNode.Type {
		case branch:
			if len(curRoute) == 0 {
				curRootHash = rootNode.Val[16]
			} else {
				curRootHash = rootNode.Val[curRoute[0]]
				curRoute = curRoute[1:]
			}
		case ext:
			path := rootNode.Val[0]
			next := rootNode.Val[1]
			matchLen := prefixLen(path, curRoute)
			if matchLen != len(path) {
				return nil, ErrNotFound
			}
			curRootHash = next
			curRoute = curRoute[matchLen:]
		case leaf:
			path := rootNode.Val[0]
			if !bytes.Equal(path, curRoute) {
				return nil, ErrNotFound
			}
			return rootNode.Val[1], nil
		default:
			return nil, ErrInvalidNodeType
		}
	}
	return nil, ErrNotFound
}

func (t *Trie) saveNode(n *node) ([]byte, error) {
	pb, err := n.toProto()
	if err != nil {
		return nil, err
	}
	b, err := proto.Marshal(pb)
	if err != nil {
		return nil, err
	}
	hash256 := sha3.Sum256(b)
	hash := hash256[:]
	t.storage.Put(hash, b)
	return hash, nil
}

func (t *Trie) update(root []byte, route []byte, value []byte) ([]byte, error) {
	if root == nil || len(root) == 0 {
		leaf := newLeafNode(route, value)
		hash, err := t.saveNode(leaf)
		if err != nil {
			return nil, err
		}
		return hash, nil
	}
	n, err := t.fetchNode(root)
	if err != nil {
		return nil, err
	}
	switch n.Type {
	case branch:
		return t.updateBranch(n, route, value)
	case ext:
		return t.updateExt(n, route, value)
	case leaf:
		return t.updateLeaf(n, route, value)
	default:
		return nil, ErrInvalidNodeType
	}
}

func (t *Trie) updateBranch(n *node, route []byte, value []byte) ([]byte, error) {
	var hash []byte
	var err error

	if len(route) == 0 {
		leaf := newLeafNode([]byte{}, value)
		if hash, err = t.saveNode(leaf); err != nil {
			return nil, err
		}
		n.Val[16] = hash
	} else {
		hash, err = t.update(n.Val[route[0]], route[1:], value)
		if err != nil {
			return nil, err
		}
		n.Val[route[0]] = hash
	}

	if hash, err = t.saveNode(n); err != nil {
		return nil, err
	}
	return hash, nil
}

func (t *Trie) updateExt(n *node, route []byte, value []byte) ([]byte, error) {
	path := n.Val[0]
	next := n.Val[1]
	matchLen := prefixLen(path, route)
	var hash []byte
	var err error
	if matchLen == len(path) { // this value will add to next branch's [16] slot
		if hash, err = t.update(next, route[matchLen:], value); err != nil {
			return nil, err
		}
		n.Val[1] = hash
		return t.saveNode(n)
	}

	branch := emptyBranchNode()
	if matchLen+1 == len(path) {
		branch.Val[path[matchLen]] = next
	} else {
		n.Val[0] = path[matchLen+1:]
		if hash, err = t.saveNode(n); err != nil {
			return nil, err
		}
		branch.Val[path[matchLen]] = hash
	}

	if matchLen == len(route) {
		leaf := newLeafNode([]byte{}, value)
		if hash, err = t.saveNode(leaf); err != nil {
			return nil, err
		}
		branch.Val[16] = hash
	} else {
		leaf := newLeafNode(route[matchLen+1:], value)
		if hash, err = t.saveNode(leaf); err != nil {
			return nil, err
		}
		branch.Val[route[matchLen]] = hash
	}

	if hash, err = t.saveNode(branch); err != nil {
		return nil, err
	}
	return t.addExtToBranch(hash, path[:matchLen])
}

func (t *Trie) updateLeaf(n *node, route []byte, value []byte) ([]byte, error) {
	path := n.Val[0]
	matchLen := prefixLen(path, route)

	if matchLen == len(route) && len(path) == len(route) { // same key
		n.Val[1] = value
		return t.saveNode(n)
	}

	var hash []byte
	var err error
	branch := emptyBranchNode()

	if matchLen == len(path) {
		leaf := newLeafNode([]byte{}, n.Val[1])
		if hash, err = t.saveNode(leaf); err != nil {
			return nil, err
		}
		branch.Val[16] = hash
	} else {
		n.Val[0] = path[matchLen+1:]
		if hash, err = t.saveNode(n); err != nil {
			return nil, err
		}
		branch.Val[path[matchLen]] = hash
	}

	if matchLen == len(route) {
		leaf := newLeafNode([]byte{}, value)
		if hash, err = t.saveNode(leaf); err != nil {
			return nil, err
		}
		branch.Val[16] = hash
	} else {
		leaf := newLeafNode(route[matchLen+1:], value)
		if hash, err = t.saveNode(leaf); err != nil {
			return nil, err
		}
		branch.Val[route[matchLen]] = hash
	}

	if hash, err = t.saveNode(branch); err != nil {
		return nil, err
	}
	return t.addExtToBranch(hash, path[:matchLen])
}

func emptyBranchNode() *node {
	return &node{
		Type: branch,
		Val:  [][]byte{nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil},
	}
}

// KeyToRoute 256 => 16 * 16
func KeyToRoute(key []byte) []byte {
	l := len(key) * 2
	var route = make([]byte, l)
	for i, b := range key {
		route[i*2] = b / 16
		route[i*2+1] = b % 16
	}
	return route
}

//RouteToKey convert route to key
func RouteToKey(route []byte) []byte {
	l := len(route) / 2
	key := make([]byte, l)
	for i := 0; i < l; i++ {
		key[i] = route[i*2]<<4 + route[i*2+1]
	}
	return key
}

func isBranchEmpty(n *node) bool {
	for _, v := range n.Val {
		if v != nil && len(v) > 0 {
			return false
		}
	}
	return true
}

func newExtNode(path []byte, next []byte) *node {
	return &node{
		Type: ext,
		Val:  [][]byte{path, next},
	}
}

func newLeafNode(path []byte, value []byte) *node {
	return &node{
		Type: leaf,
		Val:  [][]byte{path, value},
		//refCnt: 1,
	}
}

func prefixLen(a, b []byte) int {
	length := len(a)
	if len(b) < length {
		length = len(b)
	}
	for i := 0; i < length; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return length
}

//ShowPath return node's info in path
func (t *Trie) ShowPath(key []byte) []string {
	path, err := t.showPath(t.rootHash, KeyToRoute(key))
	if err != nil {
		return nil
	}
	return path
}
func (t *Trie) showPath(rootHash []byte, route []byte) ([]string, error) {

	pathToValue := make([]string, 0)

	curRootHash := rootHash
	curRoute := route
	for len(curRoute) >= 0 {
		rootNode, err := t.fetchNode(curRootHash)
		if err != nil {
			return nil, err
		}
		switch rootNode.Type {
		case branch:
			if len(curRoute) == 0 {
				pathToValue = append(pathToValue, fmt.Sprintf("Branch(-)"))
				curRootHash = rootNode.Val[16]
			} else {
				pathToValue = append(pathToValue, fmt.Sprintf("Branch(%v)", curRoute[0]))
				curRootHash = rootNode.Val[curRoute[0]]
				curRoute = curRoute[1:]
			}
		case ext:
			path := rootNode.Val[0]
			next := rootNode.Val[1]
			matchLen := prefixLen(path, curRoute)
			if matchLen != len(path) {
				return append(pathToValue, "Ext-NotFound"), ErrNotFound
			}
			pathToValue = append(pathToValue, fmt.Sprintf("Ext(%v)", curRoute[:matchLen]))
			curRootHash = next
			curRoute = curRoute[matchLen:]
		case leaf:
			path := rootNode.Val[0]
			if !bytes.Equal(path, curRoute) {
				return append(pathToValue, "Leaf-NotFound"), ErrNotFound
			}
			return append(pathToValue, fmt.Sprintf("Leaf(%v)", curRoute)), nil
		default:
			return nil, ErrInvalidNodeType
		}
	}
	return nil, ErrNotFound
}
