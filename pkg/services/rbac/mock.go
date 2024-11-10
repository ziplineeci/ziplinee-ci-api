// Code generated by MockGen. DO NOT EDIT.
// Source: service.go

// Package rbac is a generated GoMock package.
package rbac

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	api "github.com/ziplineeci/ziplinee-ci-api/pkg/api"
	contracts "github.com/ziplineeci/ziplinee-ci-contracts"
)

// MockService is a mock of Service interface.
type MockService struct {
	ctrl     *gomock.Controller
	recorder *MockServiceMockRecorder
}

// MockServiceMockRecorder is the mock recorder for MockService.
type MockServiceMockRecorder struct {
	mock *MockService
}

// NewMockService creates a new mock instance.
func NewMockService(ctrl *gomock.Controller) *MockService {
	mock := &MockService{ctrl: ctrl}
	mock.recorder = &MockServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockService) EXPECT() *MockServiceMockRecorder {
	return m.recorder
}

// CreateClient mocks base method.
func (m *MockService) CreateClient(ctx context.Context, client contracts.Client) (*contracts.Client, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateClient", ctx, client)
	ret0, _ := ret[0].(*contracts.Client)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateClient indicates an expected call of CreateClient.
func (mr *MockServiceMockRecorder) CreateClient(ctx, client interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateClient", reflect.TypeOf((*MockService)(nil).CreateClient), ctx, client)
}

// CreateGroup mocks base method.
func (m *MockService) CreateGroup(ctx context.Context, group contracts.Group) (*contracts.Group, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateGroup", ctx, group)
	ret0, _ := ret[0].(*contracts.Group)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateGroup indicates an expected call of CreateGroup.
func (mr *MockServiceMockRecorder) CreateGroup(ctx, group interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateGroup", reflect.TypeOf((*MockService)(nil).CreateGroup), ctx, group)
}

// CreateOrganization mocks base method.
func (m *MockService) CreateOrganization(ctx context.Context, organization contracts.Organization) (*contracts.Organization, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateOrganization", ctx, organization)
	ret0, _ := ret[0].(*contracts.Organization)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateOrganization indicates an expected call of CreateOrganization.
func (mr *MockServiceMockRecorder) CreateOrganization(ctx, organization interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateOrganization", reflect.TypeOf((*MockService)(nil).CreateOrganization), ctx, organization)
}

// CreateUser mocks base method.
func (m *MockService) CreateUser(ctx context.Context, user contracts.User) (*contracts.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateUser", ctx, user)
	ret0, _ := ret[0].(*contracts.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateUser indicates an expected call of CreateUser.
func (mr *MockServiceMockRecorder) CreateUser(ctx, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUser", reflect.TypeOf((*MockService)(nil).CreateUser), ctx, user)
}

// CreateUserFromIdentity mocks base method.
func (m *MockService) CreateUserFromIdentity(ctx context.Context, identity contracts.UserIdentity) (*contracts.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateUserFromIdentity", ctx, identity)
	ret0, _ := ret[0].(*contracts.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateUserFromIdentity indicates an expected call of CreateUserFromIdentity.
func (mr *MockServiceMockRecorder) CreateUserFromIdentity(ctx, identity interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUserFromIdentity", reflect.TypeOf((*MockService)(nil).CreateUserFromIdentity), ctx, identity)
}

// DeleteClient mocks base method.
func (m *MockService) DeleteClient(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteClient", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteClient indicates an expected call of DeleteClient.
func (mr *MockServiceMockRecorder) DeleteClient(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteClient", reflect.TypeOf((*MockService)(nil).DeleteClient), ctx, id)
}

// DeleteGroup mocks base method.
func (m *MockService) DeleteGroup(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteGroup", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteGroup indicates an expected call of DeleteGroup.
func (mr *MockServiceMockRecorder) DeleteGroup(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteGroup", reflect.TypeOf((*MockService)(nil).DeleteGroup), ctx, id)
}

// DeleteOrganization mocks base method.
func (m *MockService) DeleteOrganization(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteOrganization", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteOrganization indicates an expected call of DeleteOrganization.
func (mr *MockServiceMockRecorder) DeleteOrganization(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteOrganization", reflect.TypeOf((*MockService)(nil).DeleteOrganization), ctx, id)
}

// DeleteUser mocks base method.
func (m *MockService) DeleteUser(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteUser", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteUser indicates an expected call of DeleteUser.
func (mr *MockServiceMockRecorder) DeleteUser(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteUser", reflect.TypeOf((*MockService)(nil).DeleteUser), ctx, id)
}

// GetInheritedOrganizationsForUser mocks base method.
func (m *MockService) GetInheritedOrganizationsForUser(ctx context.Context, user contracts.User) ([]*contracts.Organization, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetInheritedOrganizationsForUser", ctx, user)
	ret0, _ := ret[0].([]*contracts.Organization)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetInheritedOrganizationsForUser indicates an expected call of GetInheritedOrganizationsForUser.
func (mr *MockServiceMockRecorder) GetInheritedOrganizationsForUser(ctx, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInheritedOrganizationsForUser", reflect.TypeOf((*MockService)(nil).GetInheritedOrganizationsForUser), ctx, user)
}

// GetInheritedRolesForUser mocks base method.
func (m *MockService) GetInheritedRolesForUser(ctx context.Context, user contracts.User) ([]*string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetInheritedRolesForUser", ctx, user)
	ret0, _ := ret[0].([]*string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetInheritedRolesForUser indicates an expected call of GetInheritedRolesForUser.
func (mr *MockServiceMockRecorder) GetInheritedRolesForUser(ctx, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInheritedRolesForUser", reflect.TypeOf((*MockService)(nil).GetInheritedRolesForUser), ctx, user)
}

// GetProviderByName mocks base method.
func (m *MockService) GetProviderByName(ctx context.Context, organization, name string) (*api.OAuthProvider, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProviderByName", ctx, organization, name)
	ret0, _ := ret[0].(*api.OAuthProvider)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetProviderByName indicates an expected call of GetProviderByName.
func (mr *MockServiceMockRecorder) GetProviderByName(ctx, organization, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProviderByName", reflect.TypeOf((*MockService)(nil).GetProviderByName), ctx, organization, name)
}

// GetProviders mocks base method.
func (m *MockService) GetProviders(ctx context.Context) ([]*api.OAuthProvider, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProviders", ctx)
	ret0, _ := ret[0].([]*api.OAuthProvider)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetProviders indicates an expected call of GetProviders.
func (mr *MockServiceMockRecorder) GetProviders(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProviders", reflect.TypeOf((*MockService)(nil).GetProviders), ctx)
}

// GetRoles mocks base method.
func (m *MockService) GetRoles(ctx context.Context) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRoles", ctx)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRoles indicates an expected call of GetRoles.
func (mr *MockServiceMockRecorder) GetRoles(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRoles", reflect.TypeOf((*MockService)(nil).GetRoles), ctx)
}

// GetUserByIdentity mocks base method.
func (m *MockService) GetUserByIdentity(ctx context.Context, identity contracts.UserIdentity) (*contracts.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserByIdentity", ctx, identity)
	ret0, _ := ret[0].(*contracts.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserByIdentity indicates an expected call of GetUserByIdentity.
func (mr *MockServiceMockRecorder) GetUserByIdentity(ctx, identity interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserByIdentity", reflect.TypeOf((*MockService)(nil).GetUserByIdentity), ctx, identity)
}

// UpdateClient mocks base method.
func (m *MockService) UpdateClient(ctx context.Context, client contracts.Client) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateClient", ctx, client)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateClient indicates an expected call of UpdateClient.
func (mr *MockServiceMockRecorder) UpdateClient(ctx, client interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateClient", reflect.TypeOf((*MockService)(nil).UpdateClient), ctx, client)
}

// UpdateGroup mocks base method.
func (m *MockService) UpdateGroup(ctx context.Context, group contracts.Group) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateGroup", ctx, group)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateGroup indicates an expected call of UpdateGroup.
func (mr *MockServiceMockRecorder) UpdateGroup(ctx, group interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateGroup", reflect.TypeOf((*MockService)(nil).UpdateGroup), ctx, group)
}

// UpdateOrganization mocks base method.
func (m *MockService) UpdateOrganization(ctx context.Context, organization contracts.Organization) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateOrganization", ctx, organization)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateOrganization indicates an expected call of UpdateOrganization.
func (mr *MockServiceMockRecorder) UpdateOrganization(ctx, organization interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateOrganization", reflect.TypeOf((*MockService)(nil).UpdateOrganization), ctx, organization)
}

// UpdatePipeline mocks base method.
func (m *MockService) UpdatePipeline(ctx context.Context, pipeline contracts.Pipeline) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdatePipeline", ctx, pipeline)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdatePipeline indicates an expected call of UpdatePipeline.
func (mr *MockServiceMockRecorder) UpdatePipeline(ctx, pipeline interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdatePipeline", reflect.TypeOf((*MockService)(nil).UpdatePipeline), ctx, pipeline)
}

// UpdateUser mocks base method.
func (m *MockService) UpdateUser(ctx context.Context, user contracts.User) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUser", ctx, user)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateUser indicates an expected call of UpdateUser.
func (mr *MockServiceMockRecorder) UpdateUser(ctx, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUser", reflect.TypeOf((*MockService)(nil).UpdateUser), ctx, user)
}
