// Code generated by MockGen. DO NOT EDIT.
// Source: client.go

// Package pubsubapi is a generated GoMock package.
package pubsubapi

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	ziplinee_ci_manifest "github.com/ziplineeci/ziplinee-ci-manifest"
)

// MockClient is a mock of Client interface.
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *MockClientMockRecorder
}

// MockClientMockRecorder is the mock recorder for MockClient.
type MockClientMockRecorder struct {
	mock *MockClient
}

// NewMockClient creates a new mock instance.
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	return m.recorder
}

// SubscribeToPubsubTriggers mocks base method.
func (m *MockClient) SubscribeToPubsubTriggers(ctx context.Context, manifestString string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubscribeToPubsubTriggers", ctx, manifestString)
	ret0, _ := ret[0].(error)
	return ret0
}

// SubscribeToPubsubTriggers indicates an expected call of SubscribeToPubsubTriggers.
func (mr *MockClientMockRecorder) SubscribeToPubsubTriggers(ctx, manifestString interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubscribeToPubsubTriggers", reflect.TypeOf((*MockClient)(nil).SubscribeToPubsubTriggers), ctx, manifestString)
}

// SubscribeToTopic mocks base method.
func (m *MockClient) SubscribeToTopic(ctx context.Context, projectID, topicID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubscribeToTopic", ctx, projectID, topicID)
	ret0, _ := ret[0].(error)
	return ret0
}

// SubscribeToTopic indicates an expected call of SubscribeToTopic.
func (mr *MockClientMockRecorder) SubscribeToTopic(ctx, projectID, topicID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubscribeToTopic", reflect.TypeOf((*MockClient)(nil).SubscribeToTopic), ctx, projectID, topicID)
}

// SubscriptionForTopic mocks base method.
func (m *MockClient) SubscriptionForTopic(ctx context.Context, message PubSubPushMessage) (*ziplinee_ci_manifest.ZiplineePubSubEvent, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubscriptionForTopic", ctx, message)
	ret0, _ := ret[0].(*ziplinee_ci_manifest.ZiplineePubSubEvent)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SubscriptionForTopic indicates an expected call of SubscriptionForTopic.
func (mr *MockClientMockRecorder) SubscriptionForTopic(ctx, message interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubscriptionForTopic", reflect.TypeOf((*MockClient)(nil).SubscriptionForTopic), ctx, message)
}
