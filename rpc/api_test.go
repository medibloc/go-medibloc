// Copyright 2018 The go-medibloc Authors
// This file is part of the go-medibloc library.
//
// The go-medibloc library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-medibloc library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-medibloc library. If not, see <http://www.gnu.org/licenses/>.

package rpc_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/medibloc/go-medibloc/rpc/mock_pb"
	"github.com/medibloc/go-medibloc/rpc/proto"
	"github.com/medibloc/go-medibloc/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestAPIService_GetMedState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock_rpcpb.NewMockApiServiceClient(ctrl)

	{
		expected := &rpcpb.GetMedStateResponse{Tail: "meditail"}
		client.EXPECT().GetMedState(gomock.Any(), gomock.Any()).Return(expected, nil)
		req := &rpcpb.NonParamsRequest{}
		resp, err := client.GetMedState(context.Background(), req)
		assert.Nil(t, err)
		assert.Equal(t, expected, resp)
	}

	// TODO test with datas
}

func TestAPIService_GetAccountState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock_rpcpb.NewMockApiServiceClient(ctrl)

	{
		balance, _ := util.NewUint128FromInt(11223344)
		balstr := balance.String()
		expected := &rpcpb.GetAccountStateResponse{Balance: balstr, Nonce: 1}
		client.EXPECT().GetAccountState(gomock.Any(), gomock.Any()).Return(expected, nil)
		req := &rpcpb.GetAccountStateRequest{Address: "0xabcdef"}
		resp, _ := client.GetAccountState(context.Background(), req)
		assert.Equal(t, expected, resp)
	}

	// TODO test with datas
}

func TestAdminService_SendTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock_rpcpb.NewMockAdminServiceClient(ctrl)

	{
		client.EXPECT().SendTransaction(gomock.Any(), gomock.Any()).Return(nil, errors.New("invalid tx"))
		req := &rpcpb.TransactionRequest{
			From:  "",
			To:    "",
			Value: "1",
			Nonce: 1,
			Hash:  []byte{},
			Sign:  []byte{},
		}
		_, err := client.SendTransaction(context.Background(), req)
		assert.NotNil(t, err)
	}
}
