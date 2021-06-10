// Code generated by counterfeiter. DO NOT EDIT.
package mocks

import (
	"sync"

	"github.com/trustbloc/orb/pkg/context/common"
	"github.com/trustbloc/orb/pkg/orbclient/nsprovider"
)

type ClientVersionProvider struct {
	CurrentStub        func() (common.ClientVersion, error)
	currentMutex       sync.RWMutex
	currentArgsForCall []struct {
	}
	currentReturns struct {
		result1 common.ClientVersion
		result2 error
	}
	currentReturnsOnCall map[int]struct {
		result1 common.ClientVersion
		result2 error
	}
	GetStub        func(uint64) (common.ClientVersion, error)
	getMutex       sync.RWMutex
	getArgsForCall []struct {
		arg1 uint64
	}
	getReturns struct {
		result1 common.ClientVersion
		result2 error
	}
	getReturnsOnCall map[int]struct {
		result1 common.ClientVersion
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *ClientVersionProvider) Current() (common.ClientVersion, error) {
	fake.currentMutex.Lock()
	ret, specificReturn := fake.currentReturnsOnCall[len(fake.currentArgsForCall)]
	fake.currentArgsForCall = append(fake.currentArgsForCall, struct {
	}{})
	fake.recordInvocation("Current", []interface{}{})
	fake.currentMutex.Unlock()
	if fake.CurrentStub != nil {
		return fake.CurrentStub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.currentReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *ClientVersionProvider) CurrentCallCount() int {
	fake.currentMutex.RLock()
	defer fake.currentMutex.RUnlock()
	return len(fake.currentArgsForCall)
}

func (fake *ClientVersionProvider) CurrentCalls(stub func() (common.ClientVersion, error)) {
	fake.currentMutex.Lock()
	defer fake.currentMutex.Unlock()
	fake.CurrentStub = stub
}

func (fake *ClientVersionProvider) CurrentReturns(result1 common.ClientVersion, result2 error) {
	fake.currentMutex.Lock()
	defer fake.currentMutex.Unlock()
	fake.CurrentStub = nil
	fake.currentReturns = struct {
		result1 common.ClientVersion
		result2 error
	}{result1, result2}
}

func (fake *ClientVersionProvider) CurrentReturnsOnCall(i int, result1 common.ClientVersion, result2 error) {
	fake.currentMutex.Lock()
	defer fake.currentMutex.Unlock()
	fake.CurrentStub = nil
	if fake.currentReturnsOnCall == nil {
		fake.currentReturnsOnCall = make(map[int]struct {
			result1 common.ClientVersion
			result2 error
		})
	}
	fake.currentReturnsOnCall[i] = struct {
		result1 common.ClientVersion
		result2 error
	}{result1, result2}
}

func (fake *ClientVersionProvider) Get(arg1 uint64) (common.ClientVersion, error) {
	fake.getMutex.Lock()
	ret, specificReturn := fake.getReturnsOnCall[len(fake.getArgsForCall)]
	fake.getArgsForCall = append(fake.getArgsForCall, struct {
		arg1 uint64
	}{arg1})
	fake.recordInvocation("Get", []interface{}{arg1})
	fake.getMutex.Unlock()
	if fake.GetStub != nil {
		return fake.GetStub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.getReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *ClientVersionProvider) GetCallCount() int {
	fake.getMutex.RLock()
	defer fake.getMutex.RUnlock()
	return len(fake.getArgsForCall)
}

func (fake *ClientVersionProvider) GetCalls(stub func(uint64) (common.ClientVersion, error)) {
	fake.getMutex.Lock()
	defer fake.getMutex.Unlock()
	fake.GetStub = stub
}

func (fake *ClientVersionProvider) GetArgsForCall(i int) uint64 {
	fake.getMutex.RLock()
	defer fake.getMutex.RUnlock()
	argsForCall := fake.getArgsForCall[i]
	return argsForCall.arg1
}

func (fake *ClientVersionProvider) GetReturns(result1 common.ClientVersion, result2 error) {
	fake.getMutex.Lock()
	defer fake.getMutex.Unlock()
	fake.GetStub = nil
	fake.getReturns = struct {
		result1 common.ClientVersion
		result2 error
	}{result1, result2}
}

func (fake *ClientVersionProvider) GetReturnsOnCall(i int, result1 common.ClientVersion, result2 error) {
	fake.getMutex.Lock()
	defer fake.getMutex.Unlock()
	fake.GetStub = nil
	if fake.getReturnsOnCall == nil {
		fake.getReturnsOnCall = make(map[int]struct {
			result1 common.ClientVersion
			result2 error
		})
	}
	fake.getReturnsOnCall[i] = struct {
		result1 common.ClientVersion
		result2 error
	}{result1, result2}
}

func (fake *ClientVersionProvider) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.currentMutex.RLock()
	defer fake.currentMutex.RUnlock()
	fake.getMutex.RLock()
	defer fake.getMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *ClientVersionProvider) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ nsprovider.ClientVersionProvider = new(ClientVersionProvider)