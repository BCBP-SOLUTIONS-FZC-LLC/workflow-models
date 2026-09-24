package dsl

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// CallScopeSeparator joins a callActivity's NodeID to the ID of each
// department that call brings into the calling plan.
const CallScopeSeparator = "::"

// ErrPlanTooLarge is returned by ExpandCalls when a plan holds more stages
// and calls than the budget once its calls are expanded.
var ErrPlanTooLarge = errors.New("dsl: expanded plan exceeds the stage budget")

// ExpandCalls returns a copy of c in which every CallPlan step is replaced by
// the SubWorkflowStep it stands for, with the called plan's departments cloned
// into the calling plan under call-scoped IDs (workflow_models_lib.md §2.6).
// A maxStages of zero or less disables the budget.
func ExpandCalls(c *CompiledCollaboration, maxStages int) (*CompiledCollaboration, error) {
	if c == nil {
		return nil, errors.New("dsl: no collaboration to expand")
	}
	originals := make(map[string]*CompiledPlan, len(c.Plans))
	for _, p := range c.Plans {
		if p == nil {
			continue
		}
		if _, dup := originals[p.Name]; dup {
			return nil, fmt.Errorf("dsl: two plans are named %q", p.Name)
		}
		originals[p.Name] = p
	}
	out, err := clone(c)
	if err != nil {
		return nil, err
	}
	for _, p := range out.Plans {
		if p == nil {
			continue
		}
		e := &expander{plans: originals, root: p, maxStages: maxStages}
		own := make(map[string]bool, len(p.Departments))
		for _, d := range p.Departments {
			own[d.ID] = true
			if err := e.count(len(d.Stages)); err != nil {
				return nil, err
			}
		}
		steps, err := e.steps(p.Execution.Steps, "", own, []string{p.Name})
		if err != nil {
			return nil, err
		}
		p.Execution.Steps = steps
	}
	return out, nil
}

// expander expands the calls of one plan, root, reading called plans from
// plans, which it never modifies.
type expander struct {
	plans     map[string]*CompiledPlan
	root      *CompiledPlan
	stages    int
	maxStages int
}

func (e *expander) count(n int) error {
	e.stages += n
	if e.maxStages > 0 && e.stages > e.maxStages {
		return fmt.Errorf("%w: plan %q holds more than %d stages", ErrPlanTooLarge, e.root.Name, e.maxStages)
	}
	return nil
}

// steps rewrites steps in place. They belong to a plan whose department IDs
// become root IDs by adding prefix; local holds those root IDs, the only
// departments a call made here may bind to; chain names the plans being
// expanded.
func (e *expander) steps(steps []ExecutionStep, prefix string, local map[string]bool, chain []string) ([]ExecutionStep, error) {
	for i := range steps {
		st := &steps[i]
		for j := range st.Sequential {
			st.Sequential[j] = prefix + st.Sequential[j]
		}
		for j := range st.Parallel {
			b := &st.Parallel[j]
			b.DeptID = scoped(prefix, b.DeptID)
			nested, err := e.steps(b.Steps, prefix, local, chain)
			if err != nil {
				return nil, err
			}
			b.Steps = nested
		}
		for j := range st.Exclusive {
			b := &st.Exclusive[j]
			b.Target = scoped(prefix, b.Target)
			b.RevertToDept = scoped(prefix, b.RevertToDept)
		}
		if sw := st.SubWorkflow; sw != nil {
			nested, err := e.steps(sw.Plan.Steps, prefix, local, chain)
			if err != nil {
				return nil, err
			}
			sw.Plan.Steps = nested
			scopePaths(prefix, sw.ErrorPaths, sw.TimerPaths, sw.MessagePaths)
		}
		scopePaths(prefix, nil, nil, st.MessagePaths)
		if st.CallPlan != nil {
			sw, err := e.call(st.CallPlan, prefix, local, chain)
			if err != nil {
				return nil, err
			}
			st.SubWorkflow, st.CallPlan = sw, nil
		}
	}
	return steps, nil
}

// call expands one CallPlan made from a plan whose departments carry
// callerPrefix in the root plan and are listed in local.
func (e *expander) call(cp *CallPlanStep, callerPrefix string, local map[string]bool, chain []string) (*SubWorkflowStep, error) {
	original, ok := e.plans[cp.Plan]
	if !ok {
		return nil, fmt.Errorf("dsl: call %q names plan %q, which is not in the collaboration", cp.NodeID, cp.Plan)
	}
	if slices.Contains(chain, cp.Plan) {
		return nil, fmt.Errorf("dsl: call %q closes a call cycle %s → %s", cp.NodeID, strings.Join(chain, " → "), cp.Plan)
	}
	called, err := clone(original)
	if err != nil {
		return nil, err
	}
	if err := checkCalled(cp, called); err != nil {
		return nil, err
	}
	bound, err := e.bindings(cp, called, callerPrefix, local)
	if err != nil {
		return nil, err
	}
	if err := e.count(1); err != nil {
		return nil, err
	}

	prefix := callerPrefix + cp.NodeID + CallScopeSeparator
	calledLocal := make(map[string]bool, len(called.Departments))
	for _, d := range called.Departments {
		calledLocal[prefix+d.ID] = true
		if err := e.adopt(d, prefix, bound, cp.Assignees); err != nil {
			return nil, err
		}
	}

	steps, err := e.steps(called.Execution.Steps, prefix, calledLocal, append(slices.Clone(chain), cp.Plan))
	if err != nil {
		return nil, err
	}
	sw := &SubWorkflowStep{
		NodeID:       cp.NodeID,
		Name:         cp.Name,
		Plan:         ExecutionPlan{Steps: steps},
		ErrorPaths:   cp.ErrorPaths,
		TimerPaths:   cp.TimerPaths,
		MessagePaths: cp.MessagePaths,
	}
	scopePaths(callerPrefix, sw.ErrorPaths, sw.TimerPaths, sw.MessagePaths)
	return sw, nil
}

// adopt adds a called department to the root plan under prefix, taking the
// bound calling department's label and IAM department when it has one.
func (e *expander) adopt(d DepartmentDef, prefix string, bound map[string]DepartmentDef, assignees map[string]string) error {
	if caller, ok := bound[d.ID]; ok {
		d.Label, d.IAMDepartmentID = caller.Label, caller.IAMDepartmentID
	}
	d.ID = prefix + d.ID
	if slices.ContainsFunc(e.root.Departments, func(r DepartmentDef) bool { return r.ID == d.ID }) {
		return fmt.Errorf("dsl: a call brings in department %q, which plan %q already has", d.ID, e.root.Name)
	}
	for i := range d.Stages {
		s := &d.Stages[i]
		if s.BoundaryTimer != nil {
			s.BoundaryTimer.TargetDept = scoped(prefix, s.BoundaryTimer.TargetDept)
		}
		if s.BoundaryMessage != nil {
			s.BoundaryMessage.TargetDept = scoped(prefix, s.BoundaryMessage.TargetDept)
		}
		if user, ok := assignees[s.NodeID]; ok {
			s.DefaultAssignees = []string{user}
		}
	}
	if err := e.count(len(d.Stages)); err != nil {
		return err
	}
	e.root.Departments = append(e.root.Departments, d)
	return nil
}

// checkCalled refuses what a call cannot express against the called plan: a
// department with no ID to scope, or an assignee for a task it lacks.
func checkCalled(cp *CallPlanStep, called *CompiledPlan) error {
	tasks := map[string]bool{}
	for _, d := range called.Departments {
		if d.ID == "" {
			return fmt.Errorf("dsl: call %q names plan %q, which has a department with an empty id", cp.NodeID, cp.Plan)
		}
		for _, s := range d.Stages {
			tasks[s.NodeID] = true
		}
	}
	for _, taskID := range slices.Sorted(maps.Keys(cp.Assignees)) {
		if !tasks[taskID] {
			return fmt.Errorf("dsl: call %q assigns task %q, which plan %q does not have", cp.NodeID, taskID, cp.Plan)
		}
	}
	return nil
}

// bindings resolves cp.Departments to the calling departments, already in the
// root plan, keyed by the called department's ID.
func (e *expander) bindings(cp *CallPlanStep, called *CompiledPlan, callerPrefix string, local map[string]bool) (map[string]DepartmentDef, error) {
	bound := make(map[string]DepartmentDef, len(cp.Departments))
	for _, calledID := range slices.Sorted(maps.Keys(cp.Departments)) {
		if !slices.ContainsFunc(called.Departments, func(d DepartmentDef) bool { return d.ID == calledID }) {
			return nil, fmt.Errorf("dsl: call %q binds department %q, which plan %q does not have", cp.NodeID, calledID, cp.Plan)
		}
		callerID := callerPrefix + cp.Departments[calledID]
		i := slices.IndexFunc(e.root.Departments, func(d DepartmentDef) bool { return d.ID == callerID })
		if i < 0 || !local[callerID] {
			return nil, fmt.Errorf("dsl: call %q binds department %q to %q, which the calling plan does not have", cp.NodeID, calledID, cp.Departments[calledID])
		}
		bound[calledID] = e.root.Departments[i]
	}
	return bound, nil
}

func scoped(prefix, deptID string) string {
	if deptID == "" {
		return ""
	}
	return prefix + deptID
}

func scopePaths(prefix string, errs []ErrorPath, timers []TimerPath, msgs []MessagePath) {
	for i := range errs {
		errs[i].TargetDept = scoped(prefix, errs[i].TargetDept)
	}
	for i := range timers {
		timers[i].TargetDept = scoped(prefix, timers[i].TargetDept)
	}
	for i := range msgs {
		msgs[i].TargetDept = scoped(prefix, msgs[i].TargetDept)
	}
}

// clone deep-copies v through its JSON form, which the round-trip test keeps
// complete for every pkg/dsl type.
func clone[T any](v *T) (*T, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("dsl: copy: %w", err)
	}
	out := new(T)
	if err := json.Unmarshal(b, out); err != nil {
		return nil, fmt.Errorf("dsl: copy: %w", err)
	}
	return out, nil
}
