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
	val
)

// Node in trie, three kinds,
// Branch Node [hash_0, hash_1, ..., hash_f]
// Extension Node [flag, encodedPath, next hash]
// Leaf Node [flag, encodedPath, value]
type node struct {
	Type ty
	Hash []byte
	// val branch: [17 children], ext: [path, next], value: [value]
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
	case branch: //Todo: DeRef.(기존 branch)
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
		if childrenCnt < 2 { //Todo: DeRef.(기존 branch, 기존 childNode)
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
				return t.saveNode(childNode) //
			case val: // compress branch to leaf
				if childIndex == 16 {
					return n.Val[16], nil
				}
				newExt := newExtNode([]byte{byte(childIndex)}, n.Val[childIndex])
				return t.saveNode(newExt)
			}
		}
		return t.saveNode(n) //Todo: DeRef.(기존 branch)
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
		if hash == nil {
			return nil, nil
		}

		childNode, err := t.fetchNode(hash)
		if err != nil {
			return nil, err
		}
		if childNode.Type == ext { // compress ext + ext to ext
			childNode.Val[0] = append(n.Val[0], childNode.Val[0]...)
			return t.saveNode(childNode) //Todo: DeRef.(기존 ext 지우기)
		}
		n.Val[1] = hash
		return t.saveNode(n) //Todo: DeRef.(기존 ext)

	case val:
		return nil, nil //Todo: DeRef.(기존 val)
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
	if n.Type < branch || n.Type > val {
		return nil, ErrInvalidNodeType
	}
	n.Hash = hash
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
		case val:
			if len(curRoute) != 0 {
				return nil, ErrNotFound
			}
			return rootNode.Val[0], nil
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
	n.Hash = hash
	return hash, nil
}

func (t *Trie) update(root []byte, route []byte, value []byte) ([]byte, error) {
	if len(root) == 0 {
		val := newValNode(value)
		hash, err := t.saveNode(val)
		if err != nil {
			return nil, err
		}
		if len(route) == 0 {
			return hash, nil
		}
		ext := newExtNode(route, hash)
		hash, err = t.saveNode(ext)
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
	case val:
		return t.updateVal(n, route, value)
	default:
		return nil, ErrInvalidNodeType
	}
}

func (t *Trie) updateBranch(n *node, route []byte, value []byte) ([]byte, error) {
	var hash []byte
	var err error

	if len(route) == 0 {
		val := newValNode(value) // Todo: DeRef.(기존 value node 있으면)
		if hash, err = t.saveNode(val); err != nil {
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

	if hash, err = t.saveNode(n); err != nil { // Todo: DeRef.
		return nil, err
	}
	return hash, nil
}

func (t *Trie) updateExt(n *node, route []byte, value []byte) ([]byte, error) {
	path := n.Val[0]
	next := n.Val[1]
	matchLen := prefixLen(path, route)

	var (
		hash []byte
		err  error
	)

	if matchLen == len(path) { // this value will add to next branch's [16] slot
		if hash, err = t.update(next, route[matchLen:], value); err != nil {
			return nil, err
		}
		n.Val[1] = hash
		return t.saveNode(n) // Todo: DeRef.
	}

	branch := emptyBranchNode()
	if matchLen+1 == len(path) { // Todo: DeRef. (기존 ext 노드 삭제)
		branch.Val[path[matchLen]] = next
	} else {
		n.Val[0] = path[matchLen+1:]
		if hash, err = t.saveNode(n); err != nil {
			return nil, err
		}
		branch.Val[path[matchLen]] = hash
	}

	val := newValNode(value)
	if hash, err = t.saveNode(val); err != nil {
		return nil, err
	}
	if matchLen == len(route) {
		branch.Val[16] = hash
	} else if matchLen+1 == len(route) {
		branch.Val[route[matchLen]] = hash
	} else {
		ext := newExtNode(route[matchLen+1:], hash)
		if hash, err = t.saveNode(ext); err != nil {
			return nil, err
		}
		branch.Val[route[matchLen]] = hash
	}

	if hash, err = t.saveNode(branch); err != nil {
		return nil, err
	}
	return t.addExtToBranch(hash, path[:matchLen])
}

func (t *Trie) updateVal(n *node, route []byte, value []byte) ([]byte, error) {
	newVal := newValNode(value)
	hash, err := t.saveNode(newVal)
	if err != nil {
		return nil, err
	}

	// Todo: DeRef.
	if len(route) == 0 { // same key
		return hash, nil // Todo: DeRef.
	}

	branch := emptyBranchNode()
	branch.Val[16] = n.Hash

	if len(route) == 1 {
		branch.Val[route[0]] = hash
	} else {
		newExt := newExtNode(route[1:], hash)
		hash, err = t.saveNode(newExt)
		if err != nil {
			return nil, err
		}
		branch.Val[route[0]] = hash
	}
	return t.saveNode(branch)
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

func newExtNode(path []byte, next []byte) *node {
	if len(path) == 0 {
		panic("nil path ext node cannot be generated")
	}
	return &node{
		Type: ext,
		Val:  [][]byte{path, next},
	}
}

func newValNode(value []byte) *node {
	return &node{
		Type: val,
		Val:  [][]byte{value},
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
				pathToValue = append(pathToValue, fmt.Sprintf("branch(-)"))
				curRootHash = rootNode.Val[16]
			} else {
				pathToValue = append(pathToValue, fmt.Sprintf("branch(%v)", curRoute[0]))
				curRootHash = rootNode.Val[curRoute[0]]
				curRoute = curRoute[1:]
			}
		case ext:
			path := rootNode.Val[0]
			next := rootNode.Val[1]
			matchLen := prefixLen(path, curRoute)
			if matchLen != len(path) {
				return append(pathToValue, "ext-NotFound"), ErrNotFound
			}
			pathToValue = append(pathToValue, fmt.Sprintf("ext(%v)", curRoute[:matchLen]))
			curRootHash = next
			curRoute = curRoute[matchLen:]
		case val:
			if len(curRoute) != 0 {
				return append(pathToValue, "val-NotFound"), ErrNotFound
			}
			return append(pathToValue, fmt.Sprintf("val")), nil
		default:
			return nil, ErrInvalidNodeType
		}
	}
	return nil, ErrNotFound
}
