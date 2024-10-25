// Code generated by mockery v2.46.3. DO NOT EDIT.

package mocks

import (
	http "net/http"

	mock "github.com/stretchr/testify/mock"
)

// ReleaseGetter is an autogenerated mock type for the ReleaseGetterInterface type
type ReleaseGetter struct {
	mock.Mock
}

// Get provides a mock function with given fields: _a0
func (_m *ReleaseGetter) Get(_a0 string) (*http.Response, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 *http.Response
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*http.Response, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(string) *http.Response); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*http.Response)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewReleaseGetter creates a new instance of ReleaseGetter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewReleaseGetter(t interface {
	mock.TestingT
	Cleanup(func())
}) *ReleaseGetter {
	mock := &ReleaseGetter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
