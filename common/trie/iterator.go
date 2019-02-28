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
)

// IteratorState represents the intermediate statue in iterator
type IteratorState struct {
	node  *node
	pos   int
	route []byte
}

// Iterator to traverse leaf node in a trie
type Iterator struct {
	stack []*IteratorState
	value []byte
	key   []byte
	root  *Trie
}

func validElementsInBranchNode(offset int, node *node) []int {
	var valid []int
	ty := node.Type
	if ty != branch {
		return valid
	}
	for i := offset; i < 16; i++ {
		if i == -1 {
			if node.Val[16] != nil && len(node.Val[16]) > 0 {
				valid = append(valid, -1)
			}
		} else if node.Val[i] != nil && len(node.Val[i]) > 0 {
			valid = append(valid, i)
		}
	}
	return valid
}

// Iterator return an iterator
func (t *Trie) Iterator(prefix []byte) (*Iterator, error) {
	rootHash, curRoute, err := t.getSubTrieWithMaxCommonPrefix(prefix)
	if len(rootHash) == 0 || err == ErrNotFound {
		return &Iterator{
			root:  t,
			stack: nil,
			value: nil,
			key:   nil,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	node, err := t.fetchNode(rootHash)
	if err != nil {
		return nil, err
	}

	pos := -1
	valid := validElementsInBranchNode(-1, node)
	if len(valid) > 0 {
		pos = valid[0]
	}

	return &Iterator{
		root:  t,
		stack: []*IteratorState{{node, pos, curRoute}},
		value: nil,
		key:   nil,
	}, nil
}

func (t *Trie) getSubTrieWithMaxCommonPrefix(prefix []byte) ([]byte, []byte, error) {
	curRootHash := t.rootHash
	curRoute := KeyToRoute(prefix)

	var route []byte
	for len(curRoute) > 0 {
		rootNode, err := t.fetchNode(curRootHash)
		if err != nil {
			return nil, nil, err
		}
		switch rootNode.Type {
		case branch:
			curRootHash = rootNode.Val[curRoute[0]]
			route = append(route, []byte{curRoute[0]}...)
			curRoute = curRoute[1:]
		case ext:
			path := rootNode.Val[0]
			next := rootNode.Val[1]
			matchLen := prefixLen(path, curRoute)
			if matchLen != len(path) && matchLen != len(curRoute) {
				return nil, nil, ErrNotFound
			}
			route = append(route, path...)
			curRootHash = next
			curRoute = curRoute[matchLen:]
		case val:
			if len(curRoute) != 0 {
				return nil, nil, ErrNotFound
			}
		default:
			return nil, nil, errors.New("unknown node type")
		}
	}
	return curRootHash, route, nil
}

// Keys returns all keys starting with the given prefix.
func (t *Trie) Keys(prefix []byte) ([][]byte, error) {
	keys := make([][]byte, 0)
	iter, err := t.Iterator(prefix)
	if err != nil {
		return nil, err
	}
	for {
		exist, err := iter.Next()
		if err != nil {
			return nil, err
		}
		if !exist {
			break
		}
		keys = append(keys, iter.Key())
	}
	return keys, nil
}

func (it *Iterator) push(node *node, pos int, route []byte) {
	it.stack = append(it.stack, &IteratorState{node, pos, route})
}

func (it *Iterator) pop() (*IteratorState, error) {
	size := len(it.stack)
	if size == 0 {
		return nil, errors.New("empty stack")
	}
	state := it.stack[size-1]
	it.stack = it.stack[0 : size-1]
	return state, nil
}

// Next return if there is next leaf node
func (it *Iterator) Next() (bool, error) {
	state, err := it.pop()
	if err != nil {
		return false, nil
	}
	node := state.node
	pos := state.pos
	route := state.route
	ty := node.Type
	for {
		switch ty {
		case branch:
			valid := validElementsInBranchNode(pos, node)
			if len(valid) == 0 {
				return false, errors.New("empty branch node")
			}
			if len(valid) > 1 {
				// curRoute := append(route, []byte{byte(valid[1])}...)
				it.push(node, valid[1], route)
			}
			if valid[0] == -1 {
				valid[0] = 16
			}
			route = append(route, byte(valid[0]))
			node, err = it.root.fetchNode(node.Val[valid[0]])
			if err != nil {
				return false, err
			}
			ty = node.Type
		case ext:
			route = append(route, node.Val[0]...)
			node, err = it.root.fetchNode(node.Val[1])
			if err != nil {
				return false, err
			}
			ty = node.Type
		case val:
			it.value = node.Val[0]
			it.key = route
			return true, nil
		default:
			return false, err
		}
		pos = -1
	}
}

// Key return current leaf node's key
func (it *Iterator) Key() []byte {
	return RouteToKey(it.key)
}

// Value return current leaf node's value
func (it *Iterator) Value() []byte {
	return it.value
}
