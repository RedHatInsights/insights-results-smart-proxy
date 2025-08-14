# Improve Test Coverage - Copy/Paste Prompt

Copy and paste this prompt to quickly improve test coverage in the insights-results-smart-proxy codebase:

---

## 🎯 TASK: Improve Test Coverage

I need you to improve test coverage for the insights-results-smart-proxy Go codebase following established patterns.

### STEP 1: Find Target Package
1. Run `make test` to see current coverage
2. Run `make coverage` for detailed breakdown
3. **Identify the package with lowest coverage** (prioritize 0% coverage packages)
4. Check if tests already exist: `find . -name "*_test.go" -path "*/PACKAGE/*"`

### STEP 2: Read Guidelines
**MUST READ**: `/ai-agent-rules/testing-guidelines.md` for complete patterns and examples

### STEP 3: Create Tests Following Established Patterns

**Required File Structure:**
```go
/*
Copyright © 2020 Red Hat, Inc.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
...
*/

package packagename_test

import (
    "testing"
    
    "github.com/stretchr/testify/assert"
    
    "github.com/RedHatInsights/insights-results-smart-proxy/packagename"
)
```

**Required Test Patterns:**

For **Utility/Metrics packages** (like metrics example):
```go
func TestFunction_BasicFunctionality(t *testing.T) {
    assert.NotPanics(t, func() {
        packagename.Function("test")
    }, "Function should not panic")
    
    assert.NotNil(t, packagename.Variable, "Variable should be initialized")
}

func TestFunction_ValidInput(t *testing.T) {
    result, err := packagename.Function("valid")
    assert.NoError(t, err)
    assert.NotNil(t, result)
}

func TestFunction_InvalidInput(t *testing.T) {
    result, err := packagename.Function("invalid")
    assert.Error(t, err)
    assert.Nil(t, result)
}
```

For **HTTP Handlers**:
```go
func TestHandler_ValidRequest(t *testing.T) {
    defer helpers.CleanAfterGock(t)
    
    helpers.AssertAPIRequest(t, mockServer, serverConfig, &helpers.APIRequest{
        Method:      http.MethodGet,
        Endpoint:    "/api/v1/endpoint",
        XRHIdentity: validToken,
    }, &helpers.APIResponse{
        StatusCode: http.StatusOK,
    })
}
```

For **Error Conditions**:
```go
func TestFunction_ErrorCondition(t *testing.T) {
    result, err := packagename.Function(invalidInput)
    assert.Error(t, err)
    assert.Nil(t, result)
    assert.Contains(t, err.Error(), "expected error text")
}
```

### STEP 4: Quality Checklist
- [ ] All public functions tested
- [ ] Both success and error paths covered
- [ ] No unused imports (causes build failure)
- [ ] Use `assert.NotPanics()` for function calls
- [ ] Use `helpers.CleanAfterGock(t)` for HTTP mocking
- [ ] Follow Arrange-Act-Assert pattern
- [ ] Meaningful test function names

### STEP 5: Validation
1. **Run tests**: `make test` (must pass)
2. **Check coverage**: `make coverage` (verify improvement)
3. **Target**: 90%+ for utility packages, 100% for small packages

### STEP 6: Report Results
Show before/after coverage percentages and highlight the improvement achieved.

---

**Example Success:** 
"✅ Improved `metrics` package from 0% to 100% coverage by adding tests for AddAPIMetricsWithNamespace(), metrics initialization, and counter operations."

**Need Help?** 
- Check existing test files for patterns: `ls **/*_test.go`
- Review `ai-agent-rules/testing-guidelines.md` for detailed examples
- Look at `tests/helpers` package for available utilities

---

## Quick Reference Commands

```bash
# Find low coverage packages
make test && make coverage

# Check existing tests
find . -name "*_test.go" -path "*/TARGET_PACKAGE/*"

# Run specific package tests
go test ./TARGET_PACKAGE/...

# Validate final results
make test && make coverage
```