// Code generated by mockery v2.46.3. DO NOT EDIT.

package mocks

import (
	openpgp "github.com/ProtonMail/go-crypto/openpgp"
	mock "github.com/stretchr/testify/mock"
)

// GpgEntityReader is an autogenerated mock type for the EntityReaderInterface type
type GpgEntityReader struct {
	mock.Mock
}

// GetPrivateKey provides a mock function with given fields: _a0, _a1
func (_m *GpgEntityReader) GetPrivateKey(_a0 string, _a1 string) (string, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for GetPrivateKey")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string) (string, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(string, string) string); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ReadArmoredKeyRing provides a mock function with given fields: _a0
func (_m *GpgEntityReader) ReadArmoredKeyRing(_a0 string) (openpgp.EntityList, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for ReadArmoredKeyRing")
	}

	var r0 openpgp.EntityList
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (openpgp.EntityList, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(string) openpgp.EntityList); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(openpgp.EntityList)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewGpgEntityReader creates a new instance of GpgEntityReader. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewGpgEntityReader(t interface {
	mock.TestingT
	Cleanup(func())
}) *GpgEntityReader {
	mock := &GpgEntityReader{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}