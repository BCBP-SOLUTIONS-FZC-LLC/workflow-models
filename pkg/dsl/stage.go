package dsl

// StageDef is one compiled user/service task stage within a DepartmentDef.
type StageDef struct {
	Type             string            `json:"type"`
	Activity         string            `json:"activity"`
	NodeID           string            `json:"node_id,omitempty"`
	Role             string            `json:"role"`
	DefaultAssignees []string          `json:"default_assignees,omitempty"`
	DueDate          string            `json:"due_date,omitempty"`
	FollowUpDate     string            `json:"follow_up_date,omitempty"`
	BoundaryTimer    *BoundaryTimer    `json:"boundary_timer,omitempty"`
	BoundaryMessage  *MessagePath      `json:"boundary_message,omitempty"`
	EngineNote       string            `json:"engine_note,omitempty"`
	Extras           map[string]string `json:"extras,omitempty"`
	IsZeebeUserTask  bool              `json:"is_zeebe_user_task,omitempty"`
}

// BoundaryTimer is a timer-boundary-event attached to a StageDef.
type BoundaryTimer struct {
	Duration     string `json:"duration"`
	Interrupting bool   `json:"interrupting"`
	TargetDept   string `json:"target_dept,omitempty"`
}
