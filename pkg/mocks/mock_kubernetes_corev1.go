// Code generated by MockGen. DO NOT EDIT.
// Source: k8s.io/client-go/kubernetes/typed/core/v1 (interfaces: CoreV1Interface,NamespaceInterface)

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
	v1 "k8s.io/api/core/v1"
	v10 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	v11 "k8s.io/client-go/kubernetes/typed/core/v1"
	rest "k8s.io/client-go/rest"
	reflect "reflect"
)

// MockCoreV1Interface is a mock of CoreV1Interface interface
type MockCoreV1Interface struct {
	ctrl     *gomock.Controller
	recorder *MockCoreV1InterfaceMockRecorder
}

// MockCoreV1InterfaceMockRecorder is the mock recorder for MockCoreV1Interface
type MockCoreV1InterfaceMockRecorder struct {
	mock *MockCoreV1Interface
}

// NewMockCoreV1Interface creates a new mock instance
func NewMockCoreV1Interface(ctrl *gomock.Controller) *MockCoreV1Interface {
	mock := &MockCoreV1Interface{ctrl: ctrl}
	mock.recorder = &MockCoreV1InterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockCoreV1Interface) EXPECT() *MockCoreV1InterfaceMockRecorder {
	return m.recorder
}

// ComponentStatuses mocks base method
func (m *MockCoreV1Interface) ComponentStatuses() v11.ComponentStatusInterface {
	ret := m.ctrl.Call(m, "ComponentStatuses")
	ret0, _ := ret[0].(v11.ComponentStatusInterface)
	return ret0
}

// ComponentStatuses indicates an expected call of ComponentStatuses
func (mr *MockCoreV1InterfaceMockRecorder) ComponentStatuses() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ComponentStatuses", reflect.TypeOf((*MockCoreV1Interface)(nil).ComponentStatuses))
}

// ConfigMaps mocks base method
func (m *MockCoreV1Interface) ConfigMaps(arg0 string) v11.ConfigMapInterface {
	ret := m.ctrl.Call(m, "ConfigMaps", arg0)
	ret0, _ := ret[0].(v11.ConfigMapInterface)
	return ret0
}

// ConfigMaps indicates an expected call of ConfigMaps
func (mr *MockCoreV1InterfaceMockRecorder) ConfigMaps(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConfigMaps", reflect.TypeOf((*MockCoreV1Interface)(nil).ConfigMaps), arg0)
}

// Endpoints mocks base method
func (m *MockCoreV1Interface) Endpoints(arg0 string) v11.EndpointsInterface {
	ret := m.ctrl.Call(m, "Endpoints", arg0)
	ret0, _ := ret[0].(v11.EndpointsInterface)
	return ret0
}

// Endpoints indicates an expected call of Endpoints
func (mr *MockCoreV1InterfaceMockRecorder) Endpoints(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Endpoints", reflect.TypeOf((*MockCoreV1Interface)(nil).Endpoints), arg0)
}

// Events mocks base method
func (m *MockCoreV1Interface) Events(arg0 string) v11.EventInterface {
	ret := m.ctrl.Call(m, "Events", arg0)
	ret0, _ := ret[0].(v11.EventInterface)
	return ret0
}

// Events indicates an expected call of Events
func (mr *MockCoreV1InterfaceMockRecorder) Events(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Events", reflect.TypeOf((*MockCoreV1Interface)(nil).Events), arg0)
}

// LimitRanges mocks base method
func (m *MockCoreV1Interface) LimitRanges(arg0 string) v11.LimitRangeInterface {
	ret := m.ctrl.Call(m, "LimitRanges", arg0)
	ret0, _ := ret[0].(v11.LimitRangeInterface)
	return ret0
}

// LimitRanges indicates an expected call of LimitRanges
func (mr *MockCoreV1InterfaceMockRecorder) LimitRanges(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LimitRanges", reflect.TypeOf((*MockCoreV1Interface)(nil).LimitRanges), arg0)
}

// Namespaces mocks base method
func (m *MockCoreV1Interface) Namespaces() v11.NamespaceInterface {
	ret := m.ctrl.Call(m, "Namespaces")
	ret0, _ := ret[0].(v11.NamespaceInterface)
	return ret0
}

// Namespaces indicates an expected call of Namespaces
func (mr *MockCoreV1InterfaceMockRecorder) Namespaces() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Namespaces", reflect.TypeOf((*MockCoreV1Interface)(nil).Namespaces))
}

// Nodes mocks base method
func (m *MockCoreV1Interface) Nodes() v11.NodeInterface {
	ret := m.ctrl.Call(m, "Nodes")
	ret0, _ := ret[0].(v11.NodeInterface)
	return ret0
}

// Nodes indicates an expected call of Nodes
func (mr *MockCoreV1InterfaceMockRecorder) Nodes() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Nodes", reflect.TypeOf((*MockCoreV1Interface)(nil).Nodes))
}

// PersistentVolumeClaims mocks base method
func (m *MockCoreV1Interface) PersistentVolumeClaims(arg0 string) v11.PersistentVolumeClaimInterface {
	ret := m.ctrl.Call(m, "PersistentVolumeClaims", arg0)
	ret0, _ := ret[0].(v11.PersistentVolumeClaimInterface)
	return ret0
}

// PersistentVolumeClaims indicates an expected call of PersistentVolumeClaims
func (mr *MockCoreV1InterfaceMockRecorder) PersistentVolumeClaims(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PersistentVolumeClaims", reflect.TypeOf((*MockCoreV1Interface)(nil).PersistentVolumeClaims), arg0)
}

// PersistentVolumes mocks base method
func (m *MockCoreV1Interface) PersistentVolumes() v11.PersistentVolumeInterface {
	ret := m.ctrl.Call(m, "PersistentVolumes")
	ret0, _ := ret[0].(v11.PersistentVolumeInterface)
	return ret0
}

// PersistentVolumes indicates an expected call of PersistentVolumes
func (mr *MockCoreV1InterfaceMockRecorder) PersistentVolumes() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PersistentVolumes", reflect.TypeOf((*MockCoreV1Interface)(nil).PersistentVolumes))
}

// PodTemplates mocks base method
func (m *MockCoreV1Interface) PodTemplates(arg0 string) v11.PodTemplateInterface {
	ret := m.ctrl.Call(m, "PodTemplates", arg0)
	ret0, _ := ret[0].(v11.PodTemplateInterface)
	return ret0
}

// PodTemplates indicates an expected call of PodTemplates
func (mr *MockCoreV1InterfaceMockRecorder) PodTemplates(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PodTemplates", reflect.TypeOf((*MockCoreV1Interface)(nil).PodTemplates), arg0)
}

// Pods mocks base method
func (m *MockCoreV1Interface) Pods(arg0 string) v11.PodInterface {
	ret := m.ctrl.Call(m, "Pods", arg0)
	ret0, _ := ret[0].(v11.PodInterface)
	return ret0
}

// Pods indicates an expected call of Pods
func (mr *MockCoreV1InterfaceMockRecorder) Pods(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Pods", reflect.TypeOf((*MockCoreV1Interface)(nil).Pods), arg0)
}

// RESTClient mocks base method
func (m *MockCoreV1Interface) RESTClient() rest.Interface {
	ret := m.ctrl.Call(m, "RESTClient")
	ret0, _ := ret[0].(rest.Interface)
	return ret0
}

// RESTClient indicates an expected call of RESTClient
func (mr *MockCoreV1InterfaceMockRecorder) RESTClient() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RESTClient", reflect.TypeOf((*MockCoreV1Interface)(nil).RESTClient))
}

// ReplicationControllers mocks base method
func (m *MockCoreV1Interface) ReplicationControllers(arg0 string) v11.ReplicationControllerInterface {
	ret := m.ctrl.Call(m, "ReplicationControllers", arg0)
	ret0, _ := ret[0].(v11.ReplicationControllerInterface)
	return ret0
}

// ReplicationControllers indicates an expected call of ReplicationControllers
func (mr *MockCoreV1InterfaceMockRecorder) ReplicationControllers(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReplicationControllers", reflect.TypeOf((*MockCoreV1Interface)(nil).ReplicationControllers), arg0)
}

// ResourceQuotas mocks base method
func (m *MockCoreV1Interface) ResourceQuotas(arg0 string) v11.ResourceQuotaInterface {
	ret := m.ctrl.Call(m, "ResourceQuotas", arg0)
	ret0, _ := ret[0].(v11.ResourceQuotaInterface)
	return ret0
}

// ResourceQuotas indicates an expected call of ResourceQuotas
func (mr *MockCoreV1InterfaceMockRecorder) ResourceQuotas(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResourceQuotas", reflect.TypeOf((*MockCoreV1Interface)(nil).ResourceQuotas), arg0)
}

// Secrets mocks base method
func (m *MockCoreV1Interface) Secrets(arg0 string) v11.SecretInterface {
	ret := m.ctrl.Call(m, "Secrets", arg0)
	ret0, _ := ret[0].(v11.SecretInterface)
	return ret0
}

// Secrets indicates an expected call of Secrets
func (mr *MockCoreV1InterfaceMockRecorder) Secrets(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Secrets", reflect.TypeOf((*MockCoreV1Interface)(nil).Secrets), arg0)
}

// ServiceAccounts mocks base method
func (m *MockCoreV1Interface) ServiceAccounts(arg0 string) v11.ServiceAccountInterface {
	ret := m.ctrl.Call(m, "ServiceAccounts", arg0)
	ret0, _ := ret[0].(v11.ServiceAccountInterface)
	return ret0
}

// ServiceAccounts indicates an expected call of ServiceAccounts
func (mr *MockCoreV1InterfaceMockRecorder) ServiceAccounts(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServiceAccounts", reflect.TypeOf((*MockCoreV1Interface)(nil).ServiceAccounts), arg0)
}

// Services mocks base method
func (m *MockCoreV1Interface) Services(arg0 string) v11.ServiceInterface {
	ret := m.ctrl.Call(m, "Services", arg0)
	ret0, _ := ret[0].(v11.ServiceInterface)
	return ret0
}

// Services indicates an expected call of Services
func (mr *MockCoreV1InterfaceMockRecorder) Services(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Services", reflect.TypeOf((*MockCoreV1Interface)(nil).Services), arg0)
}

// MockNamespaceInterface is a mock of NamespaceInterface interface
type MockNamespaceInterface struct {
	ctrl     *gomock.Controller
	recorder *MockNamespaceInterfaceMockRecorder
}

// MockNamespaceInterfaceMockRecorder is the mock recorder for MockNamespaceInterface
type MockNamespaceInterfaceMockRecorder struct {
	mock *MockNamespaceInterface
}

// NewMockNamespaceInterface creates a new mock instance
func NewMockNamespaceInterface(ctrl *gomock.Controller) *MockNamespaceInterface {
	mock := &MockNamespaceInterface{ctrl: ctrl}
	mock.recorder = &MockNamespaceInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockNamespaceInterface) EXPECT() *MockNamespaceInterfaceMockRecorder {
	return m.recorder
}

// Create mocks base method
func (m *MockNamespaceInterface) Create(arg0 *v1.Namespace) (*v1.Namespace, error) {
	ret := m.ctrl.Call(m, "Create", arg0)
	ret0, _ := ret[0].(*v1.Namespace)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create
func (mr *MockNamespaceInterfaceMockRecorder) Create(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockNamespaceInterface)(nil).Create), arg0)
}

// Delete mocks base method
func (m *MockNamespaceInterface) Delete(arg0 string, arg1 *v10.DeleteOptions) error {
	ret := m.ctrl.Call(m, "Delete", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete
func (mr *MockNamespaceInterfaceMockRecorder) Delete(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockNamespaceInterface)(nil).Delete), arg0, arg1)
}

// Finalize mocks base method
func (m *MockNamespaceInterface) Finalize(arg0 *v1.Namespace) (*v1.Namespace, error) {
	ret := m.ctrl.Call(m, "Finalize", arg0)
	ret0, _ := ret[0].(*v1.Namespace)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Finalize indicates an expected call of Finalize
func (mr *MockNamespaceInterfaceMockRecorder) Finalize(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Finalize", reflect.TypeOf((*MockNamespaceInterface)(nil).Finalize), arg0)
}

// Get mocks base method
func (m *MockNamespaceInterface) Get(arg0 string, arg1 v10.GetOptions) (*v1.Namespace, error) {
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(*v1.Namespace)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get
func (mr *MockNamespaceInterfaceMockRecorder) Get(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockNamespaceInterface)(nil).Get), arg0, arg1)
}

// List mocks base method
func (m *MockNamespaceInterface) List(arg0 v10.ListOptions) (*v1.NamespaceList, error) {
	ret := m.ctrl.Call(m, "List", arg0)
	ret0, _ := ret[0].(*v1.NamespaceList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List
func (mr *MockNamespaceInterfaceMockRecorder) List(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockNamespaceInterface)(nil).List), arg0)
}

// Patch mocks base method
func (m *MockNamespaceInterface) Patch(arg0 string, arg1 types.PatchType, arg2 []byte, arg3 ...string) (*v1.Namespace, error) {
	varargs := []interface{}{arg0, arg1, arg2}
	for _, a := range arg3 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Patch", varargs...)
	ret0, _ := ret[0].(*v1.Namespace)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Patch indicates an expected call of Patch
func (mr *MockNamespaceInterfaceMockRecorder) Patch(arg0, arg1, arg2 interface{}, arg3 ...interface{}) *gomock.Call {
	varargs := append([]interface{}{arg0, arg1, arg2}, arg3...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Patch", reflect.TypeOf((*MockNamespaceInterface)(nil).Patch), varargs...)
}

// Update mocks base method
func (m *MockNamespaceInterface) Update(arg0 *v1.Namespace) (*v1.Namespace, error) {
	ret := m.ctrl.Call(m, "Update", arg0)
	ret0, _ := ret[0].(*v1.Namespace)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update
func (mr *MockNamespaceInterfaceMockRecorder) Update(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockNamespaceInterface)(nil).Update), arg0)
}

// UpdateStatus mocks base method
func (m *MockNamespaceInterface) UpdateStatus(arg0 *v1.Namespace) (*v1.Namespace, error) {
	ret := m.ctrl.Call(m, "UpdateStatus", arg0)
	ret0, _ := ret[0].(*v1.Namespace)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateStatus indicates an expected call of UpdateStatus
func (mr *MockNamespaceInterfaceMockRecorder) UpdateStatus(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateStatus", reflect.TypeOf((*MockNamespaceInterface)(nil).UpdateStatus), arg0)
}

// Watch mocks base method
func (m *MockNamespaceInterface) Watch(arg0 v10.ListOptions) (watch.Interface, error) {
	ret := m.ctrl.Call(m, "Watch", arg0)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Watch indicates an expected call of Watch
func (mr *MockNamespaceInterfaceMockRecorder) Watch(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Watch", reflect.TypeOf((*MockNamespaceInterface)(nil).Watch), arg0)
}
