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
// into the calling plan under call-scoped IDs (workflow-models LLD §2.6).
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
		if err := checkBoundaries(p); err != nil {
			return nil, err
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
		steps, err := e.steps(p.Execution.Steps, "", own, []string{p.Name}, nil)
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
// expanded; down holds, by a call's NodeID, the assignees an outer call names
// for that call's tasks.
func (e *expander) steps(steps []ExecutionStep, prefix string, local map[string]bool, chain []string, down map[string]map[string]string) ([]ExecutionStep, error) {
	for i := range steps {
		st := &steps[i]
		for j := range st.Sequential {
			st.Sequential[j] = prefix + st.Sequential[j]
		}
		for j := range st.Parallel {
			b := &st.Parallel[j]
			b.DeptID = scoped(prefix, b.DeptID)
			nested, err := e.steps(b.Steps, prefix, local, chain, down)
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
			nested, err := e.steps(sw.Plan.Steps, prefix, local, chain, down)
			if err != nil {
				return nil, err
			}
			sw.Plan.Steps = nested
			scopePaths(prefix, sw.ErrorPaths, sw.TimerPaths, sw.MessagePaths)
		}
		scopePaths(prefix, nil, nil, st.MessagePaths)
		if st.CallPlan != nil {
			sw, err := e.call(st.CallPlan, prefix, local, chain, down[st.CallPlan.NodeID])
			if err != nil {
				return nil, err
			}
			st.SubWorkflow, st.CallPlan = sw, nil
		}
	}
	return steps, nil
}

// call expands one CallPlan made from a plan whose departments carry
// callerPrefix in the root plan and are listed in local. inherited is what
// outer calls name for this call's tasks, and wins over cp's own Assignees.
func (e *expander) call(cp *CallPlanStep, callerPrefix string, local map[string]bool, chain []string, inherited map[string]string) (*SubWorkflowStep, error) {
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
	direct, down := splitAssignees(cp.Assignees, inherited)
	if err := checkCalled(cp, called, direct, down); err != nil {
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
		if err := e.adopt(d, prefix, bound, direct); err != nil {
			return nil, err
		}
	}

	steps, err := e.steps(called.Execution.Steps, prefix, calledLocal, append(slices.Clone(chain), cp.Plan), down)
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

// splitAssignees merges what outer calls name (inherited) over a call's own
// Assignees, then splits the keys into the called plan's own task IDs and, by
// the NodeID of the inner call they lead through, paths to tasks further down.
func splitAssignees(own, inherited map[string]string) (direct map[string]string, down map[string]map[string]string) {
	merged := maps.Clone(own)
	if merged == nil {
		merged = map[string]string{}
	}
	maps.Copy(merged, inherited)
	direct, down = map[string]string{}, map[string]map[string]string{}
	for key, user := range merged {
		call, rest, nested := strings.Cut(key, CallScopeSeparator)
		if !nested {
			direct[key] = user
			continue
		}
		if down[call] == nil {
			down[call] = map[string]string{}
		}
		down[call][rest] = user
	}
	return direct, down
}

// checkCalled refuses what a call cannot express against the called plan: a
// department with no ID to scope, an assignee for a task it lacks, or a path
// through a call it does not make.
func checkCalled(cp *CallPlanStep, called *CompiledPlan, direct map[string]string, down map[string]map[string]string) error {
	tasks := map[string]bool{}
	for _, d := range called.Departments {
		if d.ID == "" {
			return fmt.Errorf("dsl: call %q names plan %q, which has a department with an empty id", cp.NodeID, cp.Plan)
		}
		for _, s := range d.Stages {
			tasks[s.NodeID] = true
		}
	}
	for _, taskID := range slices.Sorted(maps.Keys(direct)) {
		if !tasks[taskID] {
			return fmt.Errorf("dsl: call %q assigns task %q, which plan %q does not have", cp.NodeID, taskID, cp.Plan)
		}
	}
	calls := callNodeIDs(called.Execution.Steps, map[string]bool{})
	for _, inner := range slices.Sorted(maps.Keys(down)) {
		if !calls[inner] {
			return fmt.Errorf("dsl: call %q assigns tasks through call %q, which plan %q does not make", cp.NodeID, inner, cp.Plan)
		}
	}
	return nil
}

func callNodeIDs(steps []ExecutionStep, into map[string]bool) map[string]bool {
	for _, st := range steps {
		if st.CallPlan != nil {
			into[st.CallPlan.NodeID] = true
		}
		for _, b := range st.Parallel {
			callNodeIDs(b.Steps, into)
		}
		if st.SubWorkflow != nil {
			callNodeIDs(st.SubWorkflow.Plan.Steps, into)
		}
	}
	return into
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

// checkBoundaries refuses a boundary that neither moves to a department nor
// terminates, or does both, so an empty target is never read as either.
func checkBoundaries(p *CompiledPlan) error {
	bad := func(target string, terminates bool) bool { return (target == "") != terminates }
	for _, d := range p.Departments {
		for _, s := range d.Stages {
			if (s.BoundaryTimer != nil && bad(s.BoundaryTimer.TargetDept, s.BoundaryTimer.Terminates)) ||
				(s.BoundaryMessage != nil && bad(s.BoundaryMessage.TargetDept, s.BoundaryMessage.Terminates)) {
				return fmt.Errorf("dsl: plan %q: a boundary on stage %q must name a target department or terminate, not both or neither", p.Name, s.NodeID)
			}
		}
	}
	return checkStepBoundaries(p.Name, p.Execution.Steps, bad)
}

func checkStepBoundaries(plan string, steps []ExecutionStep, bad func(string, bool) bool) error {
	pathsOK := func(errs []ErrorPath, timers []TimerPath, msgs []MessagePath) bool {
		return !slices.ContainsFunc(errs, func(x ErrorPath) bool { return bad(x.TargetDept, x.Terminates) }) &&
			!slices.ContainsFunc(timers, func(x TimerPath) bool { return bad(x.TargetDept, x.Terminates) }) &&
			!slices.ContainsFunc(msgs, func(x MessagePath) bool { return bad(x.TargetDept, x.Terminates) })
	}
	for _, st := range steps {
		if !pathsOK(nil, nil, st.MessagePaths) {
			return fmt.Errorf("dsl: plan %q: a message boundary on step %v must name a target department or terminate, not both or neither", plan, st.Sequential)
		}
		if sw := st.SubWorkflow; sw != nil {
			if !pathsOK(sw.ErrorPaths, sw.TimerPaths, sw.MessagePaths) {
				return fmt.Errorf("dsl: plan %q: a boundary on sub_workflow %q must name a target department or terminate, not both or neither", plan, sw.NodeID)
			}
			if err := checkStepBoundaries(plan, sw.Plan.Steps, bad); err != nil {
				return err
			}
		}
		if cp := st.CallPlan; cp != nil && !pathsOK(cp.ErrorPaths, cp.TimerPaths, cp.MessagePaths) {
			return fmt.Errorf("dsl: plan %q: a boundary on call %q must name a target department or terminate, not both or neither", plan, cp.NodeID)
		}
		for _, b := range st.Parallel {
			if err := checkStepBoundaries(plan, b.Steps, bad); err != nil {
				return err
			}
		}
	}
	return nil
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
