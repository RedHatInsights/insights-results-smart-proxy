# Testing Guidelines for AI Agents

This document provides comprehensive testing rules, best practices, and patterns for improving test coverage in the insights-results-smart-proxy Go codebase.

## Testing Rules and Best Practices

### 1. IDENTIFYING GOOD TEST CANDIDATES

**Priority Order:**
- **Priority 1**: Packages with 0% coverage (highest impact, easiest wins)
- **Priority 2**: Utility packages with <50% coverage  
- **Priority 3**: Business logic functions with complex branching
- **Priority 4**: Handler functions with missing error path coverage

**How to Find Candidates:**
```bash
make test
make coverage
# Look for packages with low coverage percentages
```

### 2. ESTABLISHED PATTERNS FROM CODEBASE

#### File Structure
- Use `package packagename_test` for external testing
- Create `export_test.go` files to expose private functions for testing:
```go
package auth

var (
    GetAuthTokenHeader = getAuthTokenHeader
    ValidateToken = validateToken
)
```

#### Import Conventions
```go
import (
    "testing"
    
    "github.com/stretchr/testify/assert"
    
    "github.com/RedHatInsights/insights-results-smart-proxy/packagename"
    "github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
    ctypes "github.com/RedHatInsights/insights-results-types"
)
```

### 3. TESTING INFRASTRUCTURE USAGE

#### Test Helpers (Always Use These)
- **HTTP Testing**: `helpers.AssertAPIRequest()`, `helpers.AssertAPIv2Request()`
- **Server Setup**: `helpers.CreateHTTPServer()` for server testing
- **Mocking**: `helpers.GockExpectAPIRequest()` with `defer helpers.CleanAfterGock(t)`
- **Redis**: `helpers.GetMockRedis()` with `helpers.RedisExpectationsMet(t, server)`
- **Timeouts**: `helpers.RunTestWithTimeout()` for integration tests
- **JSON**: `helpers.ToJSONString()` for response comparisons

#### Gock for HTTP Mocking
```go
defer helpers.CleanAfterGock(t)
helpers.GockExpectAPIRequest(t, endpoint, &helpers.APIRequest{...}, &helpers.APIResponse{...})
```

### 4. TESTING PATTERNS BY CODE TYPE

#### Metrics/Utilities (Simple Functions)
```go
func TestFunction_Scenario(t *testing.T) {
    assert.NotPanics(t, func() {
        function.DoSomething()
    }, "Function should not panic")
    
    assert.NotNil(t, function.GetSomething(), "Should return non-nil value")
}

func TestFunction_ValidInput(t *testing.T) {
    // Arrange
    input := "valid_input"
    
    // Act  
    result, err := packagename.Function(input)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

#### HTTP Handlers
```go
func TestHandler_ValidRequest(t *testing.T) {
    defer helpers.CleanAfterGock(t)
    helpers.GockExpectAPIRequest(t, endpoint, &helpers.APIRequest{...}, &helpers.APIResponse{...})
    
    helpers.AssertAPIRequest(t, mockServer, serverConfig, &helpers.APIRequest{
        Method:      http.MethodGet,
        Endpoint:    "/api/v1/endpoint",
        XRHIdentity: validToken,
    }, &helpers.APIResponse{
        StatusCode: http.StatusOK,
    })
}
```

#### Authentication Testing
```go
func TestFunction_ValidAuth(t *testing.T) {
    req := httptest.NewRequest(http.MethodGet, "/", nil)
    req.Header.Set(auth.XRHAuthTokenHeader, validToken)
    
    result, err := function(req)
    assert.NoError(t, err)
    assert.NotNil(t, result)
}

func TestFunction_InvalidAuth(t *testing.T) {
    req := httptest.NewRequest(http.MethodGet, "/", nil)
    req.Header.Set(auth.XRHAuthTokenHeader, invalidToken)
    
    result, err := function(req)
    assert.Error(t, err)
    assert.Nil(t, result)
}
```

#### Error Testing
```go
func TestFunction_Error(t *testing.T) {
    result, err := function(invalidInput)
    assert.Error(t, err)
    assert.Nil(t, result)
    assert.Contains(t, err.Error(), "expected error message")
}

func TestFunction_SpecificErrorType(t *testing.T) {
    _, err := function(invalidInput)
    assert.IsType(t, &SpecificError{}, err)
}
```

#### Table-Driven Tests
```go
func TestFunction(t *testing.T) {
    testCases := []struct {
        name           string
        input          InputType
        expectedResult ExpectedType
        expectedError  string
    }{
        {"valid input", validInput, expectedResult, ""},
        {"invalid input", invalidInput, nil, "expected error"},
        {"edge case", edgeInput, edgeResult, ""},
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result, err := functionUnderTest(tc.input)
            
            if tc.expectedError != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tc.expectedError)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tc.expectedResult, result)
            }
        })
    }
}
```

### 5. REDIS TESTING PATTERNS

```go
func TestRedisFunction(t *testing.T) {
    mockRedis, server := helpers.GetMockRedis()
    defer helpers.RedisExpectationsMet(t, server)
    
    server.ExpectScan(0, "*", 0).SetVal([]string{"key1", "key2"}, 0)
    server.ExpectHMGet("key1", "field1", "field2").SetVal([]interface{}{"val1", "val2"})
    
    result, err := functionUnderTest(mockRedis)
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### 6. TESTING BEST PRACTICES

#### Test Structure (Always Follow)
```go
func TestFunction_Scenario(t *testing.T) {
    // Arrange: setup test data, mocks, etc.
    setup()
    defer cleanup()
    
    // Act: call function under test
    result, err := functionUnderTest(input)
    
    // Assert: verify results and side effects
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

#### Cleanup Patterns
```go
// Always clean up mocks
defer helpers.CleanAfterGock(t)

// Always verify Redis expectations
defer helpers.RedisExpectationsMet(t, server)

// Clean up resources
defer func() {
    // cleanup code
}()
```

#### Error Handling Patterns
```go
// Test for specific error types
assert.IsType(t, &SpecificError{}, err)

// Test for authentication errors
assert.Equal(t, auth.MissingTokenMessage, err.(*auth.AuthenticationError).ErrString)

// Test for HTTP status codes
assert.Equal(t, http.StatusForbidden, recorder.Code)
```

### 7. COMMON MISTAKES TO AVOID

- **Don't** forget to add `defer helpers.CleanAfterGock(t)`
- **Don't** leave unused imports (Go will fail the build)
- **Don't** use hardcoded values; use constants or test data
- **Don't** test implementation details; test behavior
- **Don't** create tests that depend on external services
- **Always** use meaningful test names that describe the scenario
- **Always** test both success and error paths

### 8. COVERAGE TARGETS

- **Utility packages**: 90-100% coverage
- **Small packages** (like metrics): 100% coverage
- **Handler packages**: 80-90% coverage
- **Complex business logic**: 85-95% coverage

---

## 🚀 REUSABLE PROMPT TEMPLATE FOR FUTURE TEST COVERAGE IMPROVEMENT

Copy and paste this template when working on test coverage improvement:

```markdown
# Test Coverage Improvement Task

I need to improve test coverage for a Go package in the insights-results-smart-proxy codebase.

## Current Situation
- Run `make test` and `make coverage` to identify the package with lowest coverage
- Target: Improve coverage from X% to 90%+ 

## Instructions

### 1. Analysis Phase
- Find the package with lowest test coverage (especially 0% coverage packages)
- Check if test file exists: `find . -name "*_test.go" -path "*/PACKAGE/*"`
- Examine package structure and public functions

### 2. Create Tests Following Established Patterns

**File Structure:**
```go
/*
Copyright © 2020 Red Hat, Inc.
Licensed under the Apache License, Version 2.0...
*/

package packagename_test

import (
    "testing"
    
    "github.com/stretchr/testify/assert"
    
    "github.com/RedHatInsights/insights-results-smart-proxy/packagename"
)
```

**Test Patterns:**
```go
func TestFunction_ValidInput(t *testing.T) {
    // Arrange
    input := "valid_input"
    
    // Act  
    result, err := packagename.Function(input)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
}

func TestFunction_InvalidInput(t *testing.T) {
    result, err := packagename.Function("invalid")
    assert.Error(t, err)
    assert.Nil(t, result)
}

func TestFunction_NoPanic(t *testing.T) {
    assert.NotPanics(t, func() {
        packagename.Function("test")
    }, "Function should not panic")
}
```

**For HTTP Handlers:**
```go
func TestHandler_Success(t *testing.T) {
    defer helpers.CleanAfterGock(t)
    helpers.GockExpectAPIRequest(t, endpoint, &helpers.APIRequest{...}, &helpers.APIResponse{...})
    
    helpers.AssertAPIRequest(t, mockServer, serverConfig, &helpers.APIRequest{
        Method:      http.MethodGet,
        Endpoint:    "/api/v1/endpoint",
        XRHIdentity: validToken,
    }, &helpers.APIResponse{
        StatusCode: http.StatusOK,
    })
}
```

**For Error Testing:**
```go
func TestFunction_ErrorCondition(t *testing.T) {
    result, err := packagename.Function(invalidInput)
    assert.Error(t, err)
    assert.Nil(t, result)
    assert.Contains(t, err.Error(), "expected error message")
}
```

### 3. Quality Checklist
- [ ] All public functions tested
- [ ] Error conditions covered  
- [ ] Edge cases handled
- [ ] No unused imports (will cause build failure)
- [ ] Tests pass: `make test`
- [ ] Coverage improved: `make coverage`
- [ ] Follow existing naming conventions
- [ ] Use helpers from `tests/helpers` package
- [ ] Proper cleanup with defer statements

### 4. Validation
- Run `make test` to ensure all tests pass
- Run `make coverage` to verify coverage improvement
- Target: 90%+ coverage for utility packages, 100% for small packages

## Success Criteria
- Tests pass without errors
- Coverage significantly improved (0% → 90%+)
- Code follows established patterns from existing tests
- No test flakiness or timeouts
- Proper mock cleanup and expectations met
```

---

## Last Updated

Created: 2025-08-14

This document should be updated whenever new testing patterns emerge or when significant changes are made to the testing infrastructure.