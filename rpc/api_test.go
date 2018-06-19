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

package rpc_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/medibloc/go-medibloc/rpc/mock_pb"
	"github.com/medibloc/go-medibloc/rpc/pb"
	"github.com/medibloc/go-medibloc/util"
	"github.com/stretchr/testify/assert"
)

func TestAPIService_GetMedState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock_pb.NewMockApiServiceClient(ctrl)

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
	client := mock_pb.NewMockApiServiceClient(ctrl)

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
