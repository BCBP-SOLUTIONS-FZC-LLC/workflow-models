package dsl

// ExecutionPlan is an ordered list of ExecutionStep entries describing a
// pool's or branch's control flow.
type ExecutionPlan struct {
	Steps []ExecutionStep `json:"steps"`
}

// ExecutionStep is a single control-flow node in an ExecutionPlan — exactly
// one of Sequential, Parallel, Exclusive, SubWorkflow, CallPool, or CallPlan
// is set.
type ExecutionStep struct {
	Sequential   []string          `json:"sequential,omitempty"`
	Parallel     []ParallelBranch  `json:"parallel,omitempty"`
	Exclusive    []ExclusiveBranch `json:"exclusive,omitempty"`
	SubWorkflow  *SubWorkflowStep  `json:"sub_workflow,omitempty"`
	CallPool     *CallPoolStep     `json:"call_pool,omitempty"`
	CallPlan     *CallPlanStep     `json:"call_plan,omitempty"`
	IOMapping    *IOMapping        `json:"io_mapping,omitempty"`
	Extras       map[string]string `json:"extras,omitempty"`
	MessagePaths []MessagePath     `json:"message_paths,omitempty"`
}

// MessagePath represents a message boundary event attached to a callActivity,
// subProcess, or plain userTask step. The execution service uses MessageName
// and Interrupting to route at runtime.
type MessagePath struct {
	MessageName  string `json:"message_name"`
	Interrupting bool   `json:"interrupting"`
	TargetDept   string `json:"target_dept,omitempty"`
}

// IOMapping carries variable input/output declarations: on an ExecutionStep
// for a callActivity or pool call, and on a connector-typed StageDef, where
// Inputs are resolved against the task's context at creation time and Outputs
// against the connector's own result.
type IOMapping struct {
	Inputs  []IOVar `json:"inputs,omitempty"`
	Outputs []IOVar `json:"outputs,omitempty"`
}

// IOVar maps one input or output variable between a caller and whatever it
// is calling — a sub-workflow/pool (ExecutionStep) or a connector
// (StageDef).
type IOVar struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// ParallelBranch is one concurrent branch within a parallel-gateway
// ExecutionStep.
type ParallelBranch struct {
	DeptID string          `json:"dept"`
	Steps  []ExecutionStep `json:"steps"`
}

// CallPoolStep represents a step where the main pool hands control to another
// compiled pool (Temporal child workflow). Only the main pool may emit this.
type CallPoolStep struct {
	Pool string `json:"pool"`
}

// ExclusiveBranch is one conditional branch (or back-edge revert) within an
// exclusive-gateway ExecutionStep.
type ExclusiveBranch struct {
	Target      string `json:"target,omitempty"`
	TargetStage string `json:"target_stage,omitempty"`
	// TargetNodeID is the BPMN element ID of the first user/send/receive task or
	// subprocess on this branch — used for stable machine-addressable routing.
	TargetNodeID string `json:"target_node_id,omitempty"`
	// TargetName is the BPMN element name of the target; kept for human reference.
	TargetName          string `json:"target_name,omitempty"`
	ConditionExpression string `json:"condition_expression"`
	// Terminates is true when this branch leads directly to an end event.
	Terminates bool `json:"terminates,omitempty"`
	// RevertToDept / RevertToStage / RevertToNodeID / RevertToName are set when
	// this branch is a back-edge (guarded revert/loop flow) instead of a forward branch.
	RevertToDept   string `json:"revert_to_dept,omitempty"`
	RevertToStage  string `json:"revert_to_stage,omitempty"`
	RevertToNodeID string `json:"revert_to_node_id,omitempty"`
	RevertToName   string `json:"revert_to_name,omitempty"`
}

// SubWorkflowStep represents a BPMN subprocess compiled into its own nested
// ExecutionPlan.
type SubWorkflowStep struct {
	NodeID       string        `json:"node_id,omitempty"`
	Name         string        `json:"name"`
	Plan         ExecutionPlan `json:"plan"`
	ErrorPaths   []ErrorPath   `json:"error_paths,omitempty"`
	TimerPaths   []TimerPath   `json:"timer_paths,omitempty"`
	MessagePaths []MessagePath `json:"message_paths,omitempty"`
}

// CallPlanStep calls another plan of the same collaboration: a BPMN
// callActivity whose target was compiled once into its own plan. Departments
// binds each open department of the called plan to a department of the
// calling plan; a called department absent from it keeps its own IAM
// department. Assignees gives a called task, by its NodeID, a default user
// for this call only. Consumers run ExpandCalls rather than interpreting this
// step directly.
type CallPlanStep struct {
	NodeID       string            `json:"node_id"`
	Name         string            `json:"name,omitempty"`
	Plan         string            `json:"plan"`
	Departments  map[string]string `json:"departments,omitempty"`
	Assignees    map[string]string `json:"assignees,omitempty"`
	ErrorPaths   []ErrorPath       `json:"error_paths,omitempty"`
	TimerPaths   []TimerPath       `json:"timer_paths,omitempty"`
	MessagePaths []MessagePath     `json:"message_paths,omitempty"`
}

// ErrorPath is an error-boundary-event branch attached to a SubWorkflowStep.
type ErrorPath struct {
	ErrorCode    string `json:"error_code,omitempty"`
	ShortCircuit bool   `json:"short_circuit"`
	TargetDept   string `json:"target_dept,omitempty"`
}

// TimerPath is a timer-boundary-event branch attached to a SubWorkflowStep.
type TimerPath struct {
	Duration     string `json:"duration"`
	Interrupting bool   `json:"interrupting"`
	TargetDept   string `json:"target_dept,omitempty"`
}
