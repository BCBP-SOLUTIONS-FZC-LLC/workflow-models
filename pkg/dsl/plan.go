package dsl

// CompiledPlan is one compiled BPMN pool: its departments (lanes), its
// execution plan, and visual metadata.
type CompiledPlan struct {
	Name           string             `json:"name"`
	TaskQueue      string             `json:"task_queue,omitempty"`
	Ignored        bool               `json:"ignored,omitempty"`
	Departments    []DepartmentDef    `json:"departments"`
	Execution      ExecutionPlan      `json:"execution"`
	VisualElements []VisualElementDef `json:"visual_elements,omitempty"`
}

// VisualElementDef is a non-executable BPMN visual element (e.g. a data
// store reference) kept for diagram fidelity.
type VisualElementDef struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// DepartmentDef is one compiled BPMN lane: the department's stages and its
// IAM department mapping.
type DepartmentDef struct {
	ID              string            `json:"id"`
	Label           string            `json:"label"`
	IAMDepartmentID string            `json:"iam_department_id,omitempty"`
	Ignore          bool              `json:"ignore,omitempty"`
	Props           map[string]string `json:"props,omitempty"`
	Stages          []StageDef        `json:"stages"`
}
