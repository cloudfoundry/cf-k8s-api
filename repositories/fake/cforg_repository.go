// Code generated by counterfeiter. DO NOT EDIT.
package fake

import (
	"context"
	"sync"

	"code.cloudfoundry.org/cf-k8s-api/repositories"
)

type CFOrgRepository struct {
	CreateOrgStub        func(context.Context, repositories.OrgRecord) (repositories.OrgRecord, error)
	createOrgMutex       sync.RWMutex
	createOrgArgsForCall []struct {
		arg1 context.Context
		arg2 repositories.OrgRecord
	}
	createOrgReturns struct {
		result1 repositories.OrgRecord
		result2 error
	}
	createOrgReturnsOnCall map[int]struct {
		result1 repositories.OrgRecord
		result2 error
	}
	FetchOrgsStub        func(context.Context, []string) ([]repositories.OrgRecord, error)
	fetchOrgsMutex       sync.RWMutex
	fetchOrgsArgsForCall []struct {
		arg1 context.Context
		arg2 []string
	}
	fetchOrgsReturns struct {
		result1 []repositories.OrgRecord
		result2 error
	}
	fetchOrgsReturnsOnCall map[int]struct {
		result1 []repositories.OrgRecord
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *CFOrgRepository) CreateOrg(arg1 context.Context, arg2 repositories.OrgRecord) (repositories.OrgRecord, error) {
	fake.createOrgMutex.Lock()
	ret, specificReturn := fake.createOrgReturnsOnCall[len(fake.createOrgArgsForCall)]
	fake.createOrgArgsForCall = append(fake.createOrgArgsForCall, struct {
		arg1 context.Context
		arg2 repositories.OrgRecord
	}{arg1, arg2})
	stub := fake.CreateOrgStub
	fakeReturns := fake.createOrgReturns
	fake.recordInvocation("CreateOrg", []interface{}{arg1, arg2})
	fake.createOrgMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *CFOrgRepository) CreateOrgCallCount() int {
	fake.createOrgMutex.RLock()
	defer fake.createOrgMutex.RUnlock()
	return len(fake.createOrgArgsForCall)
}

func (fake *CFOrgRepository) CreateOrgCalls(stub func(context.Context, repositories.OrgRecord) (repositories.OrgRecord, error)) {
	fake.createOrgMutex.Lock()
	defer fake.createOrgMutex.Unlock()
	fake.CreateOrgStub = stub
}

func (fake *CFOrgRepository) CreateOrgArgsForCall(i int) (context.Context, repositories.OrgRecord) {
	fake.createOrgMutex.RLock()
	defer fake.createOrgMutex.RUnlock()
	argsForCall := fake.createOrgArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *CFOrgRepository) CreateOrgReturns(result1 repositories.OrgRecord, result2 error) {
	fake.createOrgMutex.Lock()
	defer fake.createOrgMutex.Unlock()
	fake.CreateOrgStub = nil
	fake.createOrgReturns = struct {
		result1 repositories.OrgRecord
		result2 error
	}{result1, result2}
}

func (fake *CFOrgRepository) CreateOrgReturnsOnCall(i int, result1 repositories.OrgRecord, result2 error) {
	fake.createOrgMutex.Lock()
	defer fake.createOrgMutex.Unlock()
	fake.CreateOrgStub = nil
	if fake.createOrgReturnsOnCall == nil {
		fake.createOrgReturnsOnCall = make(map[int]struct {
			result1 repositories.OrgRecord
			result2 error
		})
	}
	fake.createOrgReturnsOnCall[i] = struct {
		result1 repositories.OrgRecord
		result2 error
	}{result1, result2}
}

func (fake *CFOrgRepository) FetchOrgs(arg1 context.Context, arg2 []string) ([]repositories.OrgRecord, error) {
	var arg2Copy []string
	if arg2 != nil {
		arg2Copy = make([]string, len(arg2))
		copy(arg2Copy, arg2)
	}
	fake.fetchOrgsMutex.Lock()
	ret, specificReturn := fake.fetchOrgsReturnsOnCall[len(fake.fetchOrgsArgsForCall)]
	fake.fetchOrgsArgsForCall = append(fake.fetchOrgsArgsForCall, struct {
		arg1 context.Context
		arg2 []string
	}{arg1, arg2Copy})
	stub := fake.FetchOrgsStub
	fakeReturns := fake.fetchOrgsReturns
	fake.recordInvocation("FetchOrgs", []interface{}{arg1, arg2Copy})
	fake.fetchOrgsMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *CFOrgRepository) FetchOrgsCallCount() int {
	fake.fetchOrgsMutex.RLock()
	defer fake.fetchOrgsMutex.RUnlock()
	return len(fake.fetchOrgsArgsForCall)
}

func (fake *CFOrgRepository) FetchOrgsCalls(stub func(context.Context, []string) ([]repositories.OrgRecord, error)) {
	fake.fetchOrgsMutex.Lock()
	defer fake.fetchOrgsMutex.Unlock()
	fake.FetchOrgsStub = stub
}

func (fake *CFOrgRepository) FetchOrgsArgsForCall(i int) (context.Context, []string) {
	fake.fetchOrgsMutex.RLock()
	defer fake.fetchOrgsMutex.RUnlock()
	argsForCall := fake.fetchOrgsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *CFOrgRepository) FetchOrgsReturns(result1 []repositories.OrgRecord, result2 error) {
	fake.fetchOrgsMutex.Lock()
	defer fake.fetchOrgsMutex.Unlock()
	fake.FetchOrgsStub = nil
	fake.fetchOrgsReturns = struct {
		result1 []repositories.OrgRecord
		result2 error
	}{result1, result2}
}

func (fake *CFOrgRepository) FetchOrgsReturnsOnCall(i int, result1 []repositories.OrgRecord, result2 error) {
	fake.fetchOrgsMutex.Lock()
	defer fake.fetchOrgsMutex.Unlock()
	fake.FetchOrgsStub = nil
	if fake.fetchOrgsReturnsOnCall == nil {
		fake.fetchOrgsReturnsOnCall = make(map[int]struct {
			result1 []repositories.OrgRecord
			result2 error
		})
	}
	fake.fetchOrgsReturnsOnCall[i] = struct {
		result1 []repositories.OrgRecord
		result2 error
	}{result1, result2}
}

func (fake *CFOrgRepository) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.createOrgMutex.RLock()
	defer fake.createOrgMutex.RUnlock()
	fake.fetchOrgsMutex.RLock()
	defer fake.fetchOrgsMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *CFOrgRepository) recordInvocation(key string, args []interface{}) {
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

var _ repositories.CFOrgRepository = new(CFOrgRepository)
