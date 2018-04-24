// Code generated by MockGen. DO NOT EDIT.
// Source: ./rpc.pb.go

// Package mock_rpcpb is a generated GoMock package.
package mock_rpcpb

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	proto "github.com/medibloc/go-medibloc/rpc/proto"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// MockApiServiceClient is a mock of ApiServiceClient interface
type MockApiServiceClient struct {
	ctrl     *gomock.Controller
	recorder *MockApiServiceClientMockRecorder
}

// MockApiServiceClientMockRecorder is the mock recorder for MockApiServiceClient
type MockApiServiceClientMockRecorder struct {
	mock *MockApiServiceClient
}

// NewMockApiServiceClient creates a new mock instance
func NewMockApiServiceClient(ctrl *gomock.Controller) *MockApiServiceClient {
	mock := &MockApiServiceClient{ctrl: ctrl}
	mock.recorder = &MockApiServiceClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockApiServiceClient) EXPECT() *MockApiServiceClientMockRecorder {
	return m.recorder
}

// GetAccountState mocks base method
func (m *MockApiServiceClient) GetAccountState(ctx context.Context, in *proto.GetAccountStateRequest, opts ...grpc.CallOption) (*proto.GetAccountStateResponse, error) {
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetAccountState", varargs...)
	ret0, _ := ret[0].(*proto.GetAccountStateResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAccountState indicates an expected call of GetAccountState
func (mr *MockApiServiceClientMockRecorder) GetAccountState(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAccountState", reflect.TypeOf((*MockApiServiceClient)(nil).GetAccountState), varargs...)
}

// GetMedState mocks base method
func (m *MockApiServiceClient) GetMedState(ctx context.Context, in *proto.NonParamsRequest, opts ...grpc.CallOption) (*proto.GetMedStateResponse, error) {
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetMedState", varargs...)
	ret0, _ := ret[0].(*proto.GetMedStateResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMedState indicates an expected call of GetMedState
func (mr *MockApiServiceClientMockRecorder) GetMedState(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMedState", reflect.TypeOf((*MockApiServiceClient)(nil).GetMedState), varargs...)
}

// MockApiServiceServer is a mock of ApiServiceServer interface
type MockApiServiceServer struct {
	ctrl     *gomock.Controller
	recorder *MockApiServiceServerMockRecorder
}

// MockApiServiceServerMockRecorder is the mock recorder for MockApiServiceServer
type MockApiServiceServerMockRecorder struct {
	mock *MockApiServiceServer
}

// NewMockApiServiceServer creates a new mock instance
func NewMockApiServiceServer(ctrl *gomock.Controller) *MockApiServiceServer {
	mock := &MockApiServiceServer{ctrl: ctrl}
	mock.recorder = &MockApiServiceServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockApiServiceServer) EXPECT() *MockApiServiceServerMockRecorder {
	return m.recorder
}

// GetAccountState mocks base method
func (m *MockApiServiceServer) GetAccountState(arg0 context.Context, arg1 *proto.GetAccountStateRequest) (*proto.GetAccountStateResponse, error) {
	ret := m.ctrl.Call(m, "GetAccountState", arg0, arg1)
	ret0, _ := ret[0].(*proto.GetAccountStateResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAccountState indicates an expected call of GetAccountState
func (mr *MockApiServiceServerMockRecorder) GetAccountState(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAccountState", reflect.TypeOf((*MockApiServiceServer)(nil).GetAccountState), arg0, arg1)
}

// GetMedState mocks base method
func (m *MockApiServiceServer) GetMedState(arg0 context.Context, arg1 *proto.NonParamsRequest) (*proto.GetMedStateResponse, error) {
	ret := m.ctrl.Call(m, "GetMedState", arg0, arg1)
	ret0, _ := ret[0].(*proto.GetMedStateResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMedState indicates an expected call of GetMedState
func (mr *MockApiServiceServerMockRecorder) GetMedState(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMedState", reflect.TypeOf((*MockApiServiceServer)(nil).GetMedState), arg0, arg1)
}

// MockAdminServiceClient is a mock of AdminServiceClient interface
type MockAdminServiceClient struct {
	ctrl     *gomock.Controller
	recorder *MockAdminServiceClientMockRecorder
}

// MockAdminServiceClientMockRecorder is the mock recorder for MockAdminServiceClient
type MockAdminServiceClientMockRecorder struct {
	mock *MockAdminServiceClient
}

// NewMockAdminServiceClient creates a new mock instance
func NewMockAdminServiceClient(ctrl *gomock.Controller) *MockAdminServiceClient {
	mock := &MockAdminServiceClient{ctrl: ctrl}
	mock.recorder = &MockAdminServiceClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAdminServiceClient) EXPECT() *MockAdminServiceClientMockRecorder {
	return m.recorder
}

// SendTransaction mocks base method
func (m *MockAdminServiceClient) SendTransaction(ctx context.Context, in *proto.TransactionRequest, opts ...grpc.CallOption) (*proto.TransactionResponse, error) {
	varargs := []interface{}{ctx, in}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "SendTransaction", varargs...)
	ret0, _ := ret[0].(*proto.TransactionResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SendTransaction indicates an expected call of SendTransaction
func (mr *MockAdminServiceClientMockRecorder) SendTransaction(ctx, in interface{}, opts ...interface{}) *gomock.Call {
	varargs := append([]interface{}{ctx, in}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendTransaction", reflect.TypeOf((*MockAdminServiceClient)(nil).SendTransaction), varargs...)
}

// MockAdminServiceServer is a mock of AdminServiceServer interface
type MockAdminServiceServer struct {
	ctrl     *gomock.Controller
	recorder *MockAdminServiceServerMockRecorder
}

// MockAdminServiceServerMockRecorder is the mock recorder for MockAdminServiceServer
type MockAdminServiceServerMockRecorder struct {
	mock *MockAdminServiceServer
}

// NewMockAdminServiceServer creates a new mock instance
func NewMockAdminServiceServer(ctrl *gomock.Controller) *MockAdminServiceServer {
	mock := &MockAdminServiceServer{ctrl: ctrl}
	mock.recorder = &MockAdminServiceServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAdminServiceServer) EXPECT() *MockAdminServiceServerMockRecorder {
	return m.recorder
}

// SendTransaction mocks base method
func (m *MockAdminServiceServer) SendTransaction(arg0 context.Context, arg1 *proto.TransactionRequest) (*proto.TransactionResponse, error) {
	ret := m.ctrl.Call(m, "SendTransaction", arg0, arg1)
	ret0, _ := ret[0].(*proto.TransactionResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SendTransaction indicates an expected call of SendTransaction
func (mr *MockAdminServiceServerMockRecorder) SendTransaction(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendTransaction", reflect.TypeOf((*MockAdminServiceServer)(nil).SendTransaction), arg0, arg1)
}