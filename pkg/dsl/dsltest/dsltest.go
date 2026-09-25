// Package dsltest holds golden compiled plans: bytes that
// workflow-definition-service's publish produces and that
// workflow-execution-service's tests run, so the two services test one
// contract instead of each building its own (workflow_models_lib.md §2.7).
//
// Each golden is regenerated only by definition's golden test, never by hand.
package dsltest

import (
	"bytes"
	_ "embed"
)

// LibraryCallsFile is LibraryCalls' path inside this package, for the
// definition test that regenerates it.
const LibraryCallsFile = "testdata/library_calls.json"

//go:embed testdata/library_calls.json
var libraryCalls []byte

// LibraryCalls is the stored compiled collaboration of a workflow, "Golden",
// with two lanes, Engineering and Ops, each running a call to library module
// Process_Review version 2: CA_Eng in Engineering, and CA_Ops in Ops, whose
// Assignees input gives Review_Task a default user,
// 018f2d3c-0000-7000-8000-0000000000b2.
//
// Process_Review has no lanes. Its Review_Task names user
// 550e8400-e29b-41d4-a716-446655440000, and its call Review_Check calls
// library module Process_Check version 1, whose Check_Task names no user: the
// start request must supply one. Once expanded, the module tasks' node keys
// are CA_Eng::Process_Review@v2/Review_Task,
// CA_Eng::Review_Check::Process_Check@v1/Check_Task, and the same under CA_Ops.
func LibraryCalls() []byte { return bytes.Clone(libraryCalls) }
