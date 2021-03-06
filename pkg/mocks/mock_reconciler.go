// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/docker/stacks/pkg/reconciler/reconciler (interfaces: Reconciler)

package mocks

import (
	interfaces "github.com/docker/stacks/pkg/interfaces"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockReconciler is a mock of Reconciler interface
type MockReconciler struct {
	ctrl     *gomock.Controller
	recorder *MockReconcilerMockRecorder
}

// MockReconcilerMockRecorder is the mock recorder for MockReconciler
type MockReconcilerMockRecorder struct {
	mock *MockReconciler
}

// NewMockReconciler creates a new mock instance
func NewMockReconciler(ctrl *gomock.Controller) *MockReconciler {
	mock := &MockReconciler{ctrl: ctrl}
	mock.recorder = &MockReconcilerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (_m *MockReconciler) EXPECT() *MockReconcilerMockRecorder {
	return _m.recorder
}

// Reconcile mocks base method
func (_m *MockReconciler) Reconcile(_param0 *interfaces.ReconcileResource) error {
	ret := _m.ctrl.Call(_m, "Reconcile", _param0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Reconcile indicates an expected call of Reconcile
func (_mr *MockReconcilerMockRecorder) Reconcile(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "Reconcile", reflect.TypeOf((*MockReconciler)(nil).Reconcile), arg0)
}
