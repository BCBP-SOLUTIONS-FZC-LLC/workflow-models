package dsl_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/BCBP-SOLUTIONS-FZC-LLC/workflow-models/pkg/dsl"
)

const (
	engIAM   = "018e1f2a-0000-7000-8000-00000000e001"
	opsIAM   = "018e1f2a-0000-7000-8000-00000000e002"
	legalIAM = "018e1f2a-0000-7000-8000-00000000e003"
)

func stage(nodeID string, assignees ...string) dsl.StageDef {
	return dsl.StageDef{Type: "approve", NodeID: nodeID, Activity: nodeID, Role: "reviewer", DefaultAssignees: assignees}
}

func seq(depts ...string) dsl.ExecutionStep { return dsl.ExecutionStep{Sequential: depts} }

func callStep(nodeID, plan string, depts map[string]string) dsl.ExecutionStep {
	return dsl.ExecutionStep{CallPlan: &dsl.CallPlanStep{NodeID: nodeID, Name: nodeID, Plan: plan, Departments: depts}}
}

// mainPlan has two departments, Engineering and Ops, and runs the given steps.
func mainPlan(steps ...dsl.ExecutionStep) *dsl.CompiledPlan {
	return &dsl.CompiledPlan{
		Name: "Main",
		Departments: []dsl.DepartmentDef{
			{ID: "Engineering", Label: "Engineering", IAMDepartmentID: engIAM, Stages: []dsl.StageDef{stage("Main_T1")}},
			{ID: "Ops", Label: "Operations", IAMDepartmentID: opsIAM, Stages: []dsl.StageDef{stage("Main_T2")}},
		},
		Execution: dsl.ExecutionPlan{Steps: steps},
	}
}

// reviewModule has an open slot, Sender, and a slot fixed to Legal.
func reviewModule() *dsl.CompiledPlan {
	return &dsl.CompiledPlan{
		Name: "review@v1",
		Departments: []dsl.DepartmentDef{
			{ID: "Sender", Label: "Sender", Stages: []dsl.StageDef{stage("Review_Prepare", "u-module-default")}},
			{ID: "Legal", Label: "Legal", IAMDepartmentID: legalIAM, Stages: []dsl.StageDef{stage("Review_Check")}},
		},
		Execution: dsl.ExecutionPlan{Steps: []dsl.ExecutionStep{seq("Sender", "Legal")}},
	}
}

func collab(plans ...*dsl.CompiledPlan) *dsl.CompiledCollaboration {
	return &dsl.CompiledCollaboration{MainPlan: plans[0].Name, Plans: plans, Messages: []dsl.MessageDef{}, SchemaVersion: dsl.CurrentSchemaVersion}
}

func expand(t *testing.T, c *dsl.CompiledCollaboration) *dsl.CompiledPlan {
	t.Helper()
	out, err := dsl.ExpandCalls(c, 1000)
	if err != nil {
		t.Fatalf("ExpandCalls: %v", err)
	}
	for _, p := range out.Plans {
		if p.Name == out.MainPlan {
			return p
		}
	}
	t.Fatalf("main plan %q missing after expansion", out.MainPlan)
	return nil
}

func dept(t *testing.T, p *dsl.CompiledPlan, id string) dsl.DepartmentDef {
	t.Helper()
	for _, d := range p.Departments {
		if d.ID == id {
			return d
		}
	}
	var ids []string
	for _, d := range p.Departments {
		ids = append(ids, d.ID)
	}
	t.Fatalf("department %q not in plan %q; have %v", id, p.Name, ids)
	return dsl.DepartmentDef{}
}

func subWorkflow(t *testing.T, step dsl.ExecutionStep) *dsl.SubWorkflowStep {
	t.Helper()
	if step.CallPlan != nil {
		t.Fatalf("call_plan step survived expansion: %+v", step.CallPlan)
	}
	if step.SubWorkflow == nil {
		t.Fatalf("step is not a sub_workflow: %+v", step)
	}
	return step.SubWorkflow
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestExpandCalls_NoCalls_ReturnsAnEqualCopy(t *testing.T) {
	in := collab(mainPlan(seq("Engineering", "Ops")))
	out, err := dsl.ExpandCalls(in, 1000)
	if err != nil {
		t.Fatalf("ExpandCalls: %v", err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Errorf("a plan without calls changed:\n in  %s\n out %s", mustJSON(t, in), mustJSON(t, out))
	}
	if out == in || out.Plans[0] == in.Plans[0] {
		t.Error("ExpandCalls returned the caller's own value; it must return a copy")
	}
}

func TestExpandCalls_BoundSlotTakesTheCallerDepartment(t *testing.T) {
	main := expand(t, collab(
		mainPlan(seq("Engineering"), callStep("CA_1", "review@v1", map[string]string{"Sender": "Ops"})),
		reviewModule(),
	))

	sw := subWorkflow(t, main.Execution.Steps[1])
	if sw.NodeID != "CA_1" {
		t.Errorf("sub_workflow node_id = %q, want CA_1", sw.NodeID)
	}
	if got := sw.Plan.Steps[0].Sequential; !reflect.DeepEqual(got, []string{"CA_1::Sender", "CA_1::Legal"}) {
		t.Errorf("called steps = %v, want [CA_1::Sender CA_1::Legal]", got)
	}
	sender := dept(t, main, "CA_1::Sender")
	if sender.IAMDepartmentID != opsIAM || sender.Label != "Operations" {
		t.Errorf("bound slot = {label %q, iam %q}, want Ops's {Operations, %s}", sender.Label, sender.IAMDepartmentID, opsIAM)
	}
	if len(sender.Stages) != 1 || sender.Stages[0].NodeID != "Review_Prepare" {
		t.Errorf("bound slot stages = %+v, want the module's Review_Prepare", sender.Stages)
	}
}

func TestExpandCalls_FixedSlotKeepsItsOwnDepartment(t *testing.T) {
	main := expand(t, collab(
		mainPlan(callStep("CA_1", "review@v1", map[string]string{"Sender": "Ops"})),
		reviewModule(),
	))

	legal := dept(t, main, "CA_1::Legal")
	if legal.IAMDepartmentID != legalIAM || legal.Label != "Legal" {
		t.Errorf("fixed slot = {label %q, iam %q}, want {Legal, %s}", legal.Label, legal.IAMDepartmentID, legalIAM)
	}
}

func TestExpandCalls_TwoCallSites_GetTheirOwnDepartments(t *testing.T) {
	main := expand(t, collab(
		mainPlan(
			callStep("CA_1", "review@v1", map[string]string{"Sender": "Engineering"}),
			callStep("CA_2", "review@v1", map[string]string{"Sender": "Ops"}),
		),
		reviewModule(),
	))

	first, second := dept(t, main, "CA_1::Sender"), dept(t, main, "CA_2::Sender")
	if first.IAMDepartmentID != engIAM || second.IAMDepartmentID != opsIAM {
		t.Errorf("call sites share a binding: CA_1 iam %q, CA_2 iam %q", first.IAMDepartmentID, second.IAMDepartmentID)
	}
	if got := dept(t, main, "Engineering").Stages; len(got) != 1 {
		t.Errorf("the caller's own Engineering department gained stages: %+v", got)
	}
}

func TestExpandCalls_NestedCalls_ComposeTheScope(t *testing.T) {
	outer := &dsl.CompiledPlan{
		Name: "outer@v2",
		Departments: []dsl.DepartmentDef{
			{ID: "Doer", Label: "Doer", Stages: []dsl.StageDef{stage("Outer_T1")}},
		},
		Execution: dsl.ExecutionPlan{Steps: []dsl.ExecutionStep{
			seq("Doer"),
			callStep("CA_9", "review@v1", map[string]string{"Sender": "Doer"}),
		}},
	}
	main := expand(t, collab(
		mainPlan(callStep("CA_1", "outer@v2", map[string]string{"Doer": "Engineering"})),
		outer, reviewModule(),
	))

	inner := subWorkflow(t, subWorkflow(t, main.Execution.Steps[0]).Plan.Steps[1])
	if inner.NodeID != "CA_9" {
		t.Errorf("nested sub_workflow node_id = %q, want CA_9", inner.NodeID)
	}
	if got := inner.Plan.Steps[0].Sequential; !reflect.DeepEqual(got, []string{"CA_1::CA_9::Sender", "CA_1::CA_9::Legal"}) {
		t.Errorf("nested called steps = %v", got)
	}
	if got := dept(t, main, "CA_1::CA_9::Sender").IAMDepartmentID; got != engIAM {
		t.Errorf("nested slot bound through Doer→Engineering has iam %q, want %s", got, engIAM)
	}
}

func TestExpandCalls_CallSiteAssigneesReplaceTheModuleDefault(t *testing.T) {
	call := callStep("CA_1", "review@v1", map[string]string{"Sender": "Ops"})
	call.CallPlan.Assignees = map[string]string{"Review_Prepare": "u-call-site"}
	main := expand(t, collab(mainPlan(call), reviewModule()))

	if got := dept(t, main, "CA_1::Sender").Stages[0].DefaultAssignees; !reflect.DeepEqual(got, []string{"u-call-site"}) {
		t.Errorf("Review_Prepare assignees = %v, want [u-call-site]", got)
	}
	if got := dept(t, main, "CA_1::Legal").Stages[0].DefaultAssignees; len(got) != 0 {
		t.Errorf("Review_Check assignees = %v, want none (not named by the call site)", got)
	}
}

func TestExpandCalls_RewritesEveryDepartmentReference(t *testing.T) {
	module := &dsl.CompiledPlan{
		Name: "branchy@v1",
		Departments: []dsl.DepartmentDef{
			{ID: "A", Label: "A", Stages: []dsl.StageDef{{
				Type: "approve", NodeID: "A_T1",
				BoundaryTimer:   &dsl.BoundaryTimer{Duration: "PT1H", Interrupting: true, TargetDept: "B"},
				BoundaryMessage: &dsl.MessagePath{MessageName: "stop", Interrupting: true, TargetDept: "B"},
			}}},
			{ID: "B", Label: "B", Stages: []dsl.StageDef{stage("B_T1")}},
		},
		Execution: dsl.ExecutionPlan{Steps: []dsl.ExecutionStep{
			{Parallel: []dsl.ParallelBranch{{DeptID: "A", Steps: []dsl.ExecutionStep{seq("A")}}}},
			{Exclusive: []dsl.ExclusiveBranch{
				{Target: "B", ConditionExpression: "ok"},
				{RevertToDept: "A", ConditionExpression: "redo"},
				{Terminates: true, ConditionExpression: "stop"},
			}},
			{SubWorkflow: &dsl.SubWorkflowStep{
				Name:         "inner",
				Plan:         dsl.ExecutionPlan{Steps: []dsl.ExecutionStep{seq("B")}},
				ErrorPaths:   []dsl.ErrorPath{{ErrorCode: "E", TargetDept: "A"}},
				TimerPaths:   []dsl.TimerPath{{Duration: "PT2H", TargetDept: "A"}},
				MessagePaths: []dsl.MessagePath{{MessageName: "m", TargetDept: "B"}},
			}},
			{Sequential: []string{"A"}, MessagePaths: []dsl.MessagePath{{MessageName: "n", TargetDept: "B"}}},
		}},
	}
	call := callStep("CA_1", "branchy@v1", map[string]string{"A": "Engineering", "B": "Ops"})
	call.CallPlan.ErrorPaths = []dsl.ErrorPath{{ErrorCode: "E", TargetDept: "Ops"}}
	call.CallPlan.TimerPaths = []dsl.TimerPath{{Duration: "P1D", Interrupting: true, TargetDept: "Ops"}}
	call.CallPlan.MessagePaths = []dsl.MessagePath{{MessageName: "cancel", Interrupting: true, TargetDept: "Engineering"}}
	main := expand(t, collab(mainPlan(call), module))

	sw := subWorkflow(t, main.Execution.Steps[0])
	if sw.ErrorPaths[0].TargetDept != "Ops" || sw.TimerPaths[0].TargetDept != "Ops" || sw.MessagePaths[0].TargetDept != "Engineering" {
		t.Errorf("the call's own boundary paths must keep the caller's departments: %+v %+v %+v", sw.ErrorPaths, sw.TimerPaths, sw.MessagePaths)
	}
	steps := sw.Plan.Steps
	if got := steps[0].Parallel[0]; got.DeptID != "CA_1::A" || got.Steps[0].Sequential[0] != "CA_1::A" {
		t.Errorf("parallel branch = %+v", got)
	}
	if got := steps[1].Exclusive; got[0].Target != "CA_1::B" || got[1].RevertToDept != "CA_1::A" || got[2].Target != "" {
		t.Errorf("exclusive branches = %+v", got)
	}
	inner := steps[2].SubWorkflow
	if inner.Plan.Steps[0].Sequential[0] != "CA_1::B" || inner.ErrorPaths[0].TargetDept != "CA_1::A" ||
		inner.TimerPaths[0].TargetDept != "CA_1::A" || inner.MessagePaths[0].TargetDept != "CA_1::B" {
		t.Errorf("nested sub_workflow = %+v", inner)
	}
	if got := steps[3].MessagePaths[0].TargetDept; got != "CA_1::B" {
		t.Errorf("step message path target = %q, want CA_1::B", got)
	}
	a := dept(t, main, "CA_1::A").Stages[0]
	if a.BoundaryTimer.TargetDept != "CA_1::B" || a.BoundaryMessage.TargetDept != "CA_1::B" {
		t.Errorf("stage boundary targets = %q, %q, want CA_1::B", a.BoundaryTimer.TargetDept, a.BoundaryMessage.TargetDept)
	}
}

func TestExpandCalls_RefusesBrokenCalls(t *testing.T) {
	cyclic := &dsl.CompiledPlan{Name: "loop@v1", Execution: dsl.ExecutionPlan{Steps: []dsl.ExecutionStep{callStep("CA_L", "loop@v1", nil)}}}
	cases := []struct {
		name string
		in   *dsl.CompiledCollaboration
		want string
	}{
		{"unknown plan", collab(mainPlan(callStep("CA_1", "missing@v1", nil))), "missing@v1"},
		{"cycle", collab(mainPlan(callStep("CA_1", "loop@v1", nil)), cyclic), "cycle"},
		{"binding to a department the caller lacks", collab(mainPlan(callStep("CA_1", "review@v1", map[string]string{"Sender": "Finance"})), reviewModule()), "Finance"},
		{"binding a slot the module lacks", collab(mainPlan(callStep("CA_1", "review@v1", map[string]string{"Receiver": "Ops"})), reviewModule()), "Receiver"},
		{"binding to another call's department", collab(mainPlan(
			callStep("CA_2", "review@v1", map[string]string{"Sender": "Ops"}),
			callStep("CA_1", "review@v1", map[string]string{"Sender": "CA_2::Sender"}),
		), reviewModule()), "CA_2::Sender"},
		{"a called department with an empty id", collab(mainPlan(callStep("CA_1", "blank@v1", nil)), blankModule()), "empty"},
		{"a cloned department colliding with the caller's own", collab(collidingMain(), reviewModule()), "CA_1::Legal"},
		{"assignee for a task the module lacks", collab(mainPlan(callWithAssignees("Nope")), reviewModule()), "Nope"},
		{"two plans with one name", collab(mainPlan(callStep("CA_1", "review@v1", map[string]string{"Sender": "Ops"})), reviewModule(), reviewModule()), "review@v1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := dsl.ExpandCalls(tc.in, 1000)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want an error mentioning %q", err, tc.want)
			}
		})
	}
	if _, err := dsl.ExpandCalls(nil, 1000); err == nil {
		t.Error("a nil collaboration expanded without an error")
	}
}

func blankModule() *dsl.CompiledPlan {
	return &dsl.CompiledPlan{
		Name:        "blank@v1",
		Departments: []dsl.DepartmentDef{{ID: "", Stages: []dsl.StageDef{stage("Blank_T1")}}},
		Execution:   dsl.ExecutionPlan{Steps: []dsl.ExecutionStep{seq("")}},
	}
}

func collidingMain() *dsl.CompiledPlan {
	p := mainPlan(callStep("CA_1", "review@v1", map[string]string{"Sender": "Ops"}))
	p.Departments = append(p.Departments, dsl.DepartmentDef{ID: "CA_1::Legal", Label: "CA_1::Legal"})
	return p
}

func callWithAssignees(taskID string) dsl.ExecutionStep {
	call := callStep("CA_1", "review@v1", map[string]string{"Sender": "Ops"})
	call.CallPlan.Assignees = map[string]string{taskID: "u1"}
	return call
}

func TestExpandCalls_NestedCallsBoundaryPathsTakeTheCallersScope(t *testing.T) {
	outer := &dsl.CompiledPlan{
		Name:        "outer@v2",
		Departments: []dsl.DepartmentDef{{ID: "Doer", Label: "Doer", Stages: []dsl.StageDef{stage("Outer_T1")}}},
		Execution: dsl.ExecutionPlan{Steps: []dsl.ExecutionStep{
			seq("Doer"),
			callStep("CA_9", "review@v1", map[string]string{"Sender": "Doer"}),
		}},
	}
	outer.Execution.Steps[1].CallPlan.TimerPaths = []dsl.TimerPath{{Duration: "P1D", TargetDept: "Doer"}}
	main := expand(t, collab(mainPlan(callStep("CA_1", "outer@v2", map[string]string{"Doer": "Engineering"})), outer, reviewModule()))

	inner := subWorkflow(t, subWorkflow(t, main.Execution.Steps[0]).Plan.Steps[1])
	if got := inner.TimerPaths[0].TargetDept; got != "CA_1::Doer" {
		t.Errorf("nested call's timer path target = %q, want CA_1::Doer", got)
	}
}

func TestExpandCalls_ExpandsEveryPlan(t *testing.T) {
	outer := &dsl.CompiledPlan{
		Name:        "outer@v2",
		Departments: []dsl.DepartmentDef{{ID: "Doer", Label: "Doer"}},
		Execution:   dsl.ExecutionPlan{Steps: []dsl.ExecutionStep{callStep("CA_9", "review@v1", map[string]string{"Sender": "Doer"})}},
	}
	out, err := dsl.ExpandCalls(collab(mainPlan(seq("Engineering")), outer, reviewModule()), 1000)
	if err != nil {
		t.Fatal(err)
	}
	expanded := out.Plans[1]
	subWorkflow(t, expanded.Execution.Steps[0])
	dept(t, expanded, "CA_9::Sender")
}

func TestExpandCalls_KeepsNamesAndDepartmentMetadata(t *testing.T) {
	module := reviewModule()
	module.Departments[1].Ignore = true
	module.Departments[1].Props = map[string]string{"sla": "24h"}
	call := callStep("CA_1", "review@v1", map[string]string{"Sender": "Ops"})
	call.CallPlan.Name = "Prepare response"
	main := expand(t, collab(mainPlan(call), module))

	if got := subWorkflow(t, main.Execution.Steps[0]).Name; got != "Prepare response" {
		t.Errorf("sub_workflow name = %q, want the call's name", got)
	}
	legal := dept(t, main, "CA_1::Legal")
	if !legal.Ignore || legal.Props["sla"] != "24h" {
		t.Errorf("cloned department lost ignore/props: %+v", legal)
	}
}

func TestExpandCalls_OverTheStageBudget_IsPlanTooLarge(t *testing.T) {
	in := collab(
		mainPlan(
			callStep("CA_1", "review@v1", map[string]string{"Sender": "Ops"}),
			callStep("CA_2", "review@v1", map[string]string{"Sender": "Ops"}),
		),
		reviewModule(),
	)
	// Main holds 2 stages, and each call counts itself and its 2 stages.
	if _, err := dsl.ExpandCalls(in, 8); err != nil {
		t.Fatalf("8 fits a budget of 8: %v", err)
	}
	if _, err := dsl.ExpandCalls(in, 7); !errors.Is(err, dsl.ErrPlanTooLarge) {
		t.Errorf("err = %v, want ErrPlanTooLarge", err)
	}
	if _, err := dsl.ExpandCalls(in, 0); err != nil {
		t.Errorf("a budget of 0 disables the check: %v", err)
	}
}

func TestExpandCalls_CallsCountTowardTheBudget(t *testing.T) {
	// Plans that hold only calls, each calling the next twice, hold no stages
	// but double at every level.
	plans := []*dsl.CompiledPlan{{Name: "Main", Execution: dsl.ExecutionPlan{Steps: []dsl.ExecutionStep{callStep("CA_0", "level1", nil)}}}}
	for i := 1; i <= 14; i++ {
		plans = append(plans, &dsl.CompiledPlan{Name: fmt.Sprintf("level%d", i), Execution: dsl.ExecutionPlan{Steps: []dsl.ExecutionStep{
			callStep("CA_a", fmt.Sprintf("level%d", i+1), nil),
			callStep("CA_b", fmt.Sprintf("level%d", i+1), nil),
		}}})
	}
	plans = append(plans, &dsl.CompiledPlan{Name: "level15"})
	if _, err := dsl.ExpandCalls(collab(plans...), 1000); !errors.Is(err, dsl.ErrPlanTooLarge) {
		t.Errorf("err = %v, want ErrPlanTooLarge for 2^14 calls", err)
	}
}

func TestExpandCalls_IsDeterministicAndLeavesItsInputAlone(t *testing.T) {
	build := func() *dsl.CompiledCollaboration {
		call := callStep("CA_1", "review@v1", map[string]string{"Sender": "Ops"})
		call.CallPlan.Assignees = map[string]string{"Review_Prepare": "u1", "Review_Check": "u2"}
		return collab(mainPlan(call, callStep("CA_2", "review@v1", map[string]string{"Sender": "Engineering"})), reviewModule())
	}
	in := build()
	before := mustJSON(t, in)
	first, err := dsl.ExpandCalls(in, 1000)
	if err != nil {
		t.Fatal(err)
	}
	second, err := dsl.ExpandCalls(build(), 1000)
	if err != nil {
		t.Fatal(err)
	}
	if mustJSON(t, first) != mustJSON(t, second) {
		t.Error("two expansions of the same collaboration differ")
	}
	if mustJSON(t, in) != before {
		t.Error("ExpandCalls modified its input")
	}
}
