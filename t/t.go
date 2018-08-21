package main

import (
	"encoding/hex"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/medibloc/go-medibloc/common"
	"github.com/medibloc/go-medibloc/core/pb"
)

func main() {

	// type tempType struct {
	// 	Candidates []string
	// }

	// addr, _ := hex.DecodeString("02fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c")
	// toAddr, _ := hex.DecodeString("000000000000000000000000000000000000000000000000000000000000000000")
	// val, _ := hex.DecodeString("00000000000000000000000000000000")
	// // fmt.Println(addr)
	// candidates := make([][]byte, 0)
	// candidates = append(candidates, addr)
	// candidates = append(candidates, addr)

	// a := &corepb.TransactionHashTarget{
	// 	TxType:    "vote",
	// 	From:      addr,
	// 	To:        toAddr,
	// 	Value:     val,
	// 	Timestamp: 1534401179123,
	// 	Nonce:     3,
	// 	ChainId:   1,
	// 	// Payload: &corepb.TransactionHashTarget_AddRecordPayload{
	// 	//  AddRecordPayload: &corepb.AddRecordPayload{
	// 	//    Hash: addr,
	// 	//  },
	// 	// },
	// 	Payload: &corepb.TransactionHashTarget_VotePayload{
	// 		VotePayload: &corepb.VotePayload{
	// 			Candidates: candidates,
	// 		},
	// 	},
	// }

	// r, err := proto.Marshal(a)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// // fmt.Println(a)
	// fmt.Println(proto.MarshalTextString(a))
	// fmt.Println(hex.EncodeToString(r))
	unmarshalTest()
}

func fromProto(msg proto.Message) error {
	type valueTransfer struct {
		address string
		value   int64
	}
	type test struct {
		hash    string
		payload valueTransfer
	}

	// if msg, ok := msg.(*corepb.TransactionHashTarget); ok {
	// fmt.Println(msg)
	// fmt.Println(msg.GetPayload())
	// fmt.Println(msg.GetValueTransfer())

	// switch msg.Payload.(type) {
	// case *corepb.Test_ValueTransfer:
	// 	fmt.Println("A")
	// case *corepb.Test_Vote:
	// 	fmt.Println("B")
	// }
	// }

	return nil
}

func unmarshalTest() {
	testBuffer, _ := hex.DecodeString(
		"0a2102fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c0a2102fc22ea22d02fc2469f5ec8fab44bc3de42dda2bf9ebc0c0055a9eb7df579056c")

	tx := &corepb.VotePayload{}
	if err := proto.Unmarshal(testBuffer, tx); err != nil {
		fmt.Println(err)
	}
	fmt.Println(tx)

	var candidates []common.Address
	for i, candidate := range tx.Candidates {
		fmt.Println(i)
		candidates = append(candidates, common.BytesToAddress(candidate))
	}
	fmt.Println(candidates)
}
