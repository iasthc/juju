// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/internal/worker/removal (interfaces: RemovalService,Clock)
//
// Generated by this command:
//
//	mockgen -typed -package removal -destination package_mocks_test.go github.com/juju/juju/internal/worker/removal RemovalService,Clock
//

// Package removal is a generated GoMock package.
package removal

import (
	context "context"
	reflect "reflect"
	time "time"

	clock "github.com/juju/clock"
	watcher "github.com/juju/juju/core/watcher"
	removal "github.com/juju/juju/domain/removal"
	gomock "go.uber.org/mock/gomock"
)

// MockRemovalService is a mock of RemovalService interface.
type MockRemovalService struct {
	ctrl     *gomock.Controller
	recorder *MockRemovalServiceMockRecorder
}

// MockRemovalServiceMockRecorder is the mock recorder for MockRemovalService.
type MockRemovalServiceMockRecorder struct {
	mock *MockRemovalService
}

// NewMockRemovalService creates a new mock instance.
func NewMockRemovalService(ctrl *gomock.Controller) *MockRemovalService {
	mock := &MockRemovalService{ctrl: ctrl}
	mock.recorder = &MockRemovalServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRemovalService) EXPECT() *MockRemovalServiceMockRecorder {
	return m.recorder
}

// ExecuteJob mocks base method.
func (m *MockRemovalService) ExecuteJob(arg0 context.Context, arg1 removal.Job) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExecuteJob", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// ExecuteJob indicates an expected call of ExecuteJob.
func (mr *MockRemovalServiceMockRecorder) ExecuteJob(arg0, arg1 any) *MockRemovalServiceExecuteJobCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExecuteJob", reflect.TypeOf((*MockRemovalService)(nil).ExecuteJob), arg0, arg1)
	return &MockRemovalServiceExecuteJobCall{Call: call}
}

// MockRemovalServiceExecuteJobCall wrap *gomock.Call
type MockRemovalServiceExecuteJobCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockRemovalServiceExecuteJobCall) Return(arg0 error) *MockRemovalServiceExecuteJobCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockRemovalServiceExecuteJobCall) Do(f func(context.Context, removal.Job) error) *MockRemovalServiceExecuteJobCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockRemovalServiceExecuteJobCall) DoAndReturn(f func(context.Context, removal.Job) error) *MockRemovalServiceExecuteJobCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// GetAllJobs mocks base method.
func (m *MockRemovalService) GetAllJobs(arg0 context.Context) ([]removal.Job, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllJobs", arg0)
	ret0, _ := ret[0].([]removal.Job)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAllJobs indicates an expected call of GetAllJobs.
func (mr *MockRemovalServiceMockRecorder) GetAllJobs(arg0 any) *MockRemovalServiceGetAllJobsCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllJobs", reflect.TypeOf((*MockRemovalService)(nil).GetAllJobs), arg0)
	return &MockRemovalServiceGetAllJobsCall{Call: call}
}

// MockRemovalServiceGetAllJobsCall wrap *gomock.Call
type MockRemovalServiceGetAllJobsCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockRemovalServiceGetAllJobsCall) Return(arg0 []removal.Job, arg1 error) *MockRemovalServiceGetAllJobsCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockRemovalServiceGetAllJobsCall) Do(f func(context.Context) ([]removal.Job, error)) *MockRemovalServiceGetAllJobsCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockRemovalServiceGetAllJobsCall) DoAndReturn(f func(context.Context) ([]removal.Job, error)) *MockRemovalServiceGetAllJobsCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// WatchRemovals mocks base method.
func (m *MockRemovalService) WatchRemovals() (watcher.Watcher[[]string], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchRemovals")
	ret0, _ := ret[0].(watcher.Watcher[[]string])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchRemovals indicates an expected call of WatchRemovals.
func (mr *MockRemovalServiceMockRecorder) WatchRemovals() *MockRemovalServiceWatchRemovalsCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchRemovals", reflect.TypeOf((*MockRemovalService)(nil).WatchRemovals))
	return &MockRemovalServiceWatchRemovalsCall{Call: call}
}

// MockRemovalServiceWatchRemovalsCall wrap *gomock.Call
type MockRemovalServiceWatchRemovalsCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockRemovalServiceWatchRemovalsCall) Return(arg0 watcher.Watcher[[]string], arg1 error) *MockRemovalServiceWatchRemovalsCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockRemovalServiceWatchRemovalsCall) Do(f func() (watcher.Watcher[[]string], error)) *MockRemovalServiceWatchRemovalsCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockRemovalServiceWatchRemovalsCall) DoAndReturn(f func() (watcher.Watcher[[]string], error)) *MockRemovalServiceWatchRemovalsCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// MockClock is a mock of Clock interface.
type MockClock struct {
	ctrl     *gomock.Controller
	recorder *MockClockMockRecorder
}

// MockClockMockRecorder is the mock recorder for MockClock.
type MockClockMockRecorder struct {
	mock *MockClock
}

// NewMockClock creates a new mock instance.
func NewMockClock(ctrl *gomock.Controller) *MockClock {
	mock := &MockClock{ctrl: ctrl}
	mock.recorder = &MockClockMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClock) EXPECT() *MockClockMockRecorder {
	return m.recorder
}

// NewTimer mocks base method.
func (m *MockClock) NewTimer(arg0 time.Duration) clock.Timer {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewTimer", arg0)
	ret0, _ := ret[0].(clock.Timer)
	return ret0
}

// NewTimer indicates an expected call of NewTimer.
func (mr *MockClockMockRecorder) NewTimer(arg0 any) *MockClockNewTimerCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewTimer", reflect.TypeOf((*MockClock)(nil).NewTimer), arg0)
	return &MockClockNewTimerCall{Call: call}
}

// MockClockNewTimerCall wrap *gomock.Call
type MockClockNewTimerCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockClockNewTimerCall) Return(arg0 clock.Timer) *MockClockNewTimerCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockClockNewTimerCall) Do(f func(time.Duration) clock.Timer) *MockClockNewTimerCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockClockNewTimerCall) DoAndReturn(f func(time.Duration) clock.Timer) *MockClockNewTimerCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// Now mocks base method.
func (m *MockClock) Now() time.Time {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Now")
	ret0, _ := ret[0].(time.Time)
	return ret0
}

// Now indicates an expected call of Now.
func (mr *MockClockMockRecorder) Now() *MockClockNowCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Now", reflect.TypeOf((*MockClock)(nil).Now))
	return &MockClockNowCall{Call: call}
}

// MockClockNowCall wrap *gomock.Call
type MockClockNowCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockClockNowCall) Return(arg0 time.Time) *MockClockNowCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockClockNowCall) Do(f func() time.Time) *MockClockNowCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockClockNowCall) DoAndReturn(f func() time.Time) *MockClockNowCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
