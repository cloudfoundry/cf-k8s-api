// Code generated by counterfeiter. DO NOT EDIT.
package fake

import (
	"sync"

	"code.cloudfoundry.org/cf-k8s-api/repositories/authorization"
)

type IdentityInspector struct {
	WhoAmIStub        func(string) (string, error)
	whoAmIMutex       sync.RWMutex
	whoAmIArgsForCall []struct {
		arg1 string
	}
	whoAmIReturns struct {
		result1 string
		result2 error
	}
	whoAmIReturnsOnCall map[int]struct {
		result1 string
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *IdentityInspector) WhoAmI(arg1 string) (string, error) {
	fake.whoAmIMutex.Lock()
	ret, specificReturn := fake.whoAmIReturnsOnCall[len(fake.whoAmIArgsForCall)]
	fake.whoAmIArgsForCall = append(fake.whoAmIArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.WhoAmIStub
	fakeReturns := fake.whoAmIReturns
	fake.recordInvocation("WhoAmI", []interface{}{arg1})
	fake.whoAmIMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *IdentityInspector) WhoAmICallCount() int {
	fake.whoAmIMutex.RLock()
	defer fake.whoAmIMutex.RUnlock()
	return len(fake.whoAmIArgsForCall)
}

func (fake *IdentityInspector) WhoAmICalls(stub func(string) (string, error)) {
	fake.whoAmIMutex.Lock()
	defer fake.whoAmIMutex.Unlock()
	fake.WhoAmIStub = stub
}

func (fake *IdentityInspector) WhoAmIArgsForCall(i int) string {
	fake.whoAmIMutex.RLock()
	defer fake.whoAmIMutex.RUnlock()
	argsForCall := fake.whoAmIArgsForCall[i]
	return argsForCall.arg1
}

func (fake *IdentityInspector) WhoAmIReturns(result1 string, result2 error) {
	fake.whoAmIMutex.Lock()
	defer fake.whoAmIMutex.Unlock()
	fake.WhoAmIStub = nil
	fake.whoAmIReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *IdentityInspector) WhoAmIReturnsOnCall(i int, result1 string, result2 error) {
	fake.whoAmIMutex.Lock()
	defer fake.whoAmIMutex.Unlock()
	fake.WhoAmIStub = nil
	if fake.whoAmIReturnsOnCall == nil {
		fake.whoAmIReturnsOnCall = make(map[int]struct {
			result1 string
			result2 error
		})
	}
	fake.whoAmIReturnsOnCall[i] = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *IdentityInspector) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.whoAmIMutex.RLock()
	defer fake.whoAmIMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *IdentityInspector) recordInvocation(key string, args []interface{}) {
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

var _ authorization.IdentityInspector = new(IdentityInspector)
