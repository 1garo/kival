# Implementation Plan for Milestone 9.1 - Log Package Tests

## Overview

**Objective**: Add comprehensive test coverage for the `log` package as part of Milestone 9.1
**Package**: `github.com/1garo/kival/log`
**Current Test File**: `/home/hungaroxd/dev/kival/log/log_test.go`
**Target**: ≥ 90% line coverage with complete edge case handling

## Current State Analysis

### Existing Infrastructure
- **Main Package**: `/home/hungaroxd/dev/kival/log/log.go` (243 lines)
- **Current Tests**: `/home/hungaroxd/dev/kival/log/log_test.go` (118 lines)
- **Test Framework**: Go standard testing + `github.com/stretchr/testify`
- **Test Patterns**: Uses `require.NoError()`, `assert.ErrorIs()`, `t.Helper()`, `t.Cleanup()`

### Current Test Coverage (118 lines)
```go
✅ TestLog_ReadAt_ReturnsAppendedValue
✅ TestLog_ReadAt_MultipleRecords  
✅ TestLog_ReadAt_InvalidPositionReturnsError
✅ TestLog_ReadAt_AfterCloseReturnsError
✅ TestLog_ReadAt_TruncatedRecordReturnsError
✅ TestLog_Append_InsertRecord
✅ TestLog_Append_ReadOnlySegmentError
```

### Critical Gaps Identified
1. **File Management**: No tests for `New()`, `Open()`, `parseFileID()`
2. **Capacity Handling**: No tests for `ErrCapacityExceeded` scenarios
3. **Index Building**: No tests for `BuildIndex()` method (critical functionality)
4. **Error Recovery**: Limited error path testing
5. **Integration**: No multi-file or complex scenario tests

## Implementation Plan

### Phase 1: File Management Tests (Priority: HIGH)

#### 1.1 Log File Creation Tests
```go
func TestNew_LogFileCreation(t *testing.T)
func TestNew_DirectoryCreation(t *testing.T)  
func TestNew_FilePermissions(t *testing.T)
```
**Coverage**: `New()` function, directory creation, file permissions
**Test Data**: Temporary directories, permission checks

#### 1.2 Log Opening Tests
```go
func TestOpen_EmptyDirectory(t *testing.T)
func TestOpen_ExistingFiles(t *testing.T)
func TestOpen_MultipleFilesSorted(t *testing.T)
func TestOpen_NonexistentDirectory(t *testing.T)
```
**Coverage**: `Open()` function, file discovery, sorting by ID
**Test Data**: Pre-created `.data` files, empty directories

#### 1.3 File ID Handling Tests
```go
func TestParseFileID_ValidNames(t *testing.T)
func TestParseFileID_InvalidNames(t *testing.T)
```
**Coverage**: `parseFileID()` function edge cases

### Phase 2: Capacity and Rotation Tests (Priority: HIGH)

#### 2.1 Capacity Boundary Tests
```go
func TestAppend_CapacityExceeded(t *testing.T)
func TestAppend_ExactlyAtCapacity(t *testing.T)
func TestAppend_MultipleRecordsUntilFull(t *testing.T)
```
**Coverage**: `ErrCapacityExceeded`, `MaxDataFileSize` boundary
**Test Data**: Records sized to hit capacity limits

#### 2.2 Read-Only State Tests
```go
func TestMarkReadOnly_PreventsAppend(t *testing.T)
func TestMarkReadOnly_AllowsReads(t *testing.T)
func TestMarkReadOnly_StatePersistence(t *testing.T)
```
**Coverage**: `MarkReadOnly()`, state transitions

### Phase 3: Index Building Tests (Priority: CRITICAL)

#### 3.1 Happy Path Tests
```go
func TestBuildIndex_ValidRecords(t *testing.T)
func TestBuildIndex_WithTombstones(t *testing.T)
func TestBuildIndex_MultipleKeys(t *testing.T)
func TestBuildIndex_UpdateWritePosition(t *testing.T)
```
**Coverage**: `BuildIndex()` method, index updates, tombstone handling
**Test Data**: Mixed valid/tombstone records, various key/value sizes

#### 3.2 Error Recovery Tests
```go
func TestBuildIndex_CorruptRecordHandling(t *testing.T)
func TestBuildIndex_PartialWriteHandling(t *testing.T)
func TestBuildIndex_EmptyFile(t *testing.T)
func TestBuildIndex_FileReadError(t *testing.T)
```
**Coverage**: Error scenarios, graceful failure, corruption detection
**Test Data**: Corrupted files, partial writes, empty files

#### 3.3 Index State Tests
```go
func TestBuildIndex_OverwriteExistingKeys(t *testing.T)
func TestBuildIndex_RemovesTombstonedKeys(t *testing.T)
func TestBuildIndex_LogPositionAccuracy(t *testing.T)
```
**Coverage**: Index state management, key lifecycle

### Phase 4: Error Handling Edge Cases (Priority: MEDIUM)

#### 4.1 State Management Tests
```go
func TestReadAt_ClosedFile(t *testing.T)
func TestAppend_ReadOnlyFile(t *testing.T)
func TestClose_MultipleCalls(t *testing.T)
func TestSize_AfterOperations(t *testing.T)
```
**Coverage**: Error paths, state validation, method behavior

#### 4.2 Position Handling Tests
```go
func TestReadAt_InvalidOffset(t *testing.T)
func TestReadAt_BeyondFileSize(t *testing.T)
func TestLogPosition_Construction(t *testing.T)
func TestNewLogPosition_Validation(t *testing.T)
```
**Coverage**: `LogPosition` validation, boundary conditions

### Phase 5: Integration Tests (Priority: MEDIUM)

#### 5.1 Multi-File Integration Tests
```go
func TestOpen_RebuildIndexAcrossMultipleFiles(t *testing.T)
func TestBuildIndex_MultipleFilesConsistency(t *testing.T)
func TestLogPosition_FileIDAccuracy(t *testing.T)
```
**Coverage**: Cross-file operations, consistency validation

#### 5.2 Complex Scenario Tests
```go
func TestLog_FullWorkflow(t *testing.T)
func TestLog_RecoveryScenario(t *testing.T)
func TestLog_CapacityRotationWorkflow(t *testing.T)
```
**Coverage**: End-to-end workflows, realistic usage patterns

### Phase 6: Performance/Concurrent Tests (Priority: LOW)

#### 6.1 Concurrency Tests
```go
func TestConcurrentAppends(t *testing.T)
func TestConcurrentReads(t *testing.T)
func TestConcurrentMixedOperations(t *testing.T)
```
**Coverage**: Thread safety, race condition detection

## Test Utilities and Helpers

### Required Helper Functions
```go
// File creation helpers
func createTestLogFileWithRecords(t *testing.T, id uint32, dir string, records []testRecord) *logFile
func createCorruptedLogFile(t *testing.T, id uint32, dir string, corruptionType string) *logFile
func simulatePartialWrite(t *testing.T, id uint32, dir string, partialBytes int) *logFile

// Test data generators  
func generateTestRecords(count int) []testRecord
func generateTombstoneRecords(keys []string) []testRecord
func generateMixedRecords(count int) []testRecord

// Assertion helpers
func assertLogPosition(t *testing.T, expected, actual log.LogPosition)
func assertIndexState(t *testing.T, index map[string]log.LogPosition, expectedKeys []string)
```

### Test Data Structures
```go
type testRecord struct {
    Key   []byte
    Value []byte
    IsTombstone bool
    Timestamp uint32
}

type testScenario struct {
    Name string
    Records []testRecord
    ExpectedError error
    ExpectedIndex map[string]log.LogPosition
}
```

## Implementation Strategy

### Week 1: Core Infrastructure (Phases 1-2)
1. **File Management Tests**: Implement basic `New()`, `Open()`, `parseFileID()` tests
2. **Capacity Tests**: Implement capacity boundary and read-only state tests
3. **Test Utilities**: Build helper functions for file creation and test data

### Week 2: Index Building (Phase 3) 
1. **Happy Path Tests**: Implement successful index building scenarios
2. **Error Recovery Tests**: Implement corruption and partial write handling
3. **Index State Tests**: Implement index accuracy and lifecycle tests

### Week 3: Edge Cases and Integration (Phases 4-5)
1. **Error Handling**: Implement comprehensive error path tests
2. **Integration Tests**: Implement multi-file and workflow tests
3. **Coverage Analysis**: Review and fill remaining gaps

### Week 4: Performance and Polish (Phase 6)
1. **Concurrency Tests**: Add thread safety tests if time permits
2. **Final Review**: Ensure all tests pass, coverage goals met
3. **Documentation**: Add test documentation and examples

## Success Criteria

### Coverage Metrics
- **Line Coverage**: ≥ 90% for `log.go`
- **Function Coverage**: 100% of exported functions
- **Branch Coverage**: ≥ 85% of conditional branches
- **Error Path Coverage**: 100% of error returns tested

### Quality Standards
- **Test Naming**: Follow existing `TestPackage_Function_Scenario` pattern
- **Assertions**: Use testify helpers consistently
- **Cleanup**: Proper resource cleanup with `t.Cleanup()`
- **Documentation**: Clear test descriptions and comments

### Performance Standards
- **Test Execution**: All tests complete within 10 seconds
- **Memory Usage**: No memory leaks in test execution
- **Concurrency**: No race conditions in concurrent tests

## Files to Modify

### Primary Changes
- **Extend**: `/home/hungaroxd/dev/kival/log/log_test.go`
  - Add ~25-30 new test functions
  - Add helper utilities and test data structures
  - Organize tests by functionality with clear sections

### No Changes Required
- `log.go` (implementation is complete and stable)
- Other packages (focus is specifically on log package)

## Risk Assessment

### Technical Risks
- **File System Dependencies**: Tests rely on temp directories and file operations
- **Timing Issues**: Concurrent tests may be flaky on CI/CD
- **Capacity Constants**: Hard-coded `MaxDataFileSize` may affect test reliability

### Mitigation Strategies
- **Isolation**: Use `t.TempDir()` for complete test isolation
- **Determinism**: Avoid timing-dependent assertions
- **Flexibility**: Make capacity tests work with current `MaxDataFileSize = 500`

## Dependencies

### External Dependencies
- `github.com/stretchr/testify` (already in go.mod)
- Go standard library `testing`, `os`, `path/filepath`

### Internal Dependencies  
- `github.com/1garo/kival/log` (package under test)
- `github.com/1garo/kival/record` (for record creation/validation)

## Timeline

**Total Estimated Time**: 2-3 weeks
**Milestone Deadline**: TBD (aligned with project roadmap)
**Weekly Checkpoints**: End of each phase for review

## Next Steps

1. **Review and Approve**: Get approval on this detailed plan
2. **Environment Setup**: Ensure test environment is ready
3. **Begin Implementation**: Start with Phase 1 (File Management Tests)
4. **Regular Reviews**: Weekly progress reviews and adjustments

---

*This plan focuses specifically on the log package as requested in Milestone 9.1, providing comprehensive test coverage while maintaining alignment with existing code patterns and testing standards.*