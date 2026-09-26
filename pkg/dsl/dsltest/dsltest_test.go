package dsltest_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/BCBP-SOLUTIONS-FZC-LLC/workflow-models/pkg/dsl"
	"github.com/BCBP-SOLUTIONS-FZC-LLC/workflow-models/pkg/dsl/dsltest"
)

const (
	engIAM    = "2200a5ae-2466-54f7-a1d8-243106e05705"
	opsIAM    = "92fd92cf-4b2c-58dc-bf98-f3d0af729876"
	reviewer  = "018f2d3c-0000-7000-8000-0000000000a1"
	opsReview = "018f2d3c-0000-7000-8000-0000000000b2"
	engCheck  = "018f2d3c-0000-7000-8000-0000000000c1"
	opsCheck  = "018f2d3c-0000-7000-8000-0000000000c2"
)

// stage finds the stage whose node key is key, returning its department.
func stage(plan *dsl.CompiledPlan, key string) (*dsl.DepartmentDef, *dsl.StageDef) {
	dept, node, _ := strings.Cut(key, "/")
	for i := range plan.Departments {
		d := &plan.Departments[i]
		if d.ID != dept {
			continue
		}
		for j := range d.Stages {
			if d.Stages[j].NodeID == node {
				return d, &d.Stages[j]
			}
		}
	}
	return nil, nil
}

// TestLibraryCalls pins what execution relies on in the golden: it decodes,
// it expands, and each call site runs its own copy of the module's tasks in
// the caller's department with the right default people, including a task
// two modules down that the workflow's call names by path.
func TestLibraryCalls(t *testing.T) {
	var c dsl.CompiledCollaboration
	if err := json.Unmarshal(dsltest.LibraryCalls(), &c); err != nil {
		t.Fatalf("golden does not decode: %v", err)
	}
	if c.SchemaVersion != dsl.CurrentSchemaVersion || c.MainPlan != "Golden" {
		t.Fatalf("schema_version %d, main_plan %q; want %d and Golden", c.SchemaVersion, c.MainPlan, dsl.CurrentSchemaVersion)
	}
	expanded, err := dsl.ExpandCalls(&c, 0)
	if err != nil {
		t.Fatalf("golden does not expand: %v", err)
	}
	main := expanded.Plans[0]
	if main.Name != "Golden" {
		t.Fatalf("first plan %q, want the main plan", main.Name)
	}

	for _, tc := range []struct {
		key, iam string
		defaults []string
	}{
		{"CA_Eng::Process_Review/Review_Task", engIAM, []string{reviewer}},
		{"CA_Eng::Review_Check::Process_Check/Check_Task", engIAM, []string{engCheck}},
		{"CA_Ops::Process_Review/Review_Task", opsIAM, []string{opsReview}},
		{"CA_Ops::Review_Check::Process_Check/Check_Task", opsIAM, []string{opsCheck}},
	} {
		dept, st := stage(main, tc.key)
		if st == nil {
			t.Errorf("no stage with node key %s", tc.key)
			continue
		}
		if dept.IAMDepartmentID != tc.iam {
			t.Errorf("%s runs in IAM department %s, want the caller lane's %s", tc.key, dept.IAMDepartmentID, tc.iam)
		}
		if !reflect.DeepEqual(st.DefaultAssignees, tc.defaults) {
			t.Errorf("%s defaults to %v, want %v", tc.key, st.DefaultAssignees, tc.defaults)
		}
	}
	_, review := stage(main, "CA_Eng::Process_Review/Review_Task")
	if review == nil || review.BoundaryMessage == nil ||
		*review.BoundaryMessage != (dsl.MessagePath{MessageName: "review-recalled", Interrupting: true, Terminates: true}) {
		t.Errorf("Review_Task's message boundary = %+v, want the module's own review-recalled, terminating", review)
	}
}

// TestLibraryCallBoundaries pins the boundary paths execution runs once the
// golden's calls are expanded: each call keeps its interrupting boundary,
// which terminates, and the send task runs between them.
func TestLibraryCallBoundaries(t *testing.T) {
	var c dsl.CompiledCollaboration
	if err := json.Unmarshal(dsltest.LibraryCallBoundaries(), &c); err != nil {
		t.Fatalf("golden does not decode: %v", err)
	}
	expanded, err := dsl.ExpandCalls(&c, 0)
	if err != nil {
		t.Fatalf("golden does not expand: %v", err)
	}
	steps := expanded.Plans[0].Execution.Steps
	if len(steps) != 3 || steps[0].SubWorkflow == nil || len(steps[1].Sequential) != 1 || steps[2].SubWorkflow == nil {
		t.Fatalf("steps = %+v, want CA_Timer, the send task's department, CA_Message", steps)
	}
	timer, msg := steps[0].SubWorkflow, steps[2].SubWorkflow
	if timer.NodeID != "CA_Timer" || !reflect.DeepEqual(timer.TimerPaths, []dsl.TimerPath{{Duration: "P5D", Interrupting: true, Terminates: true}}) {
		t.Errorf("CA_Timer = %s with timer paths %+v, want one interrupting P5D path that terminates", timer.NodeID, timer.TimerPaths)
	}
	if msg.NodeID != "CA_Message" || !reflect.DeepEqual(msg.MessagePaths, []dsl.MessagePath{{MessageName: "withdraw", Interrupting: true, Terminates: true}}) {
		t.Errorf("CA_Message = %s with message paths %+v, want one interrupting withdraw path that terminates", msg.NodeID, msg.MessagePaths)
	}
	_, send := stage(expanded.Plans[0], steps[1].Sequential[0]+"/Send_Withdraw")
	if send == nil || send.Extras["message"] != "withdraw" {
		t.Errorf("send task = %+v, want Send_Withdraw sending withdraw", send)
	}
}
