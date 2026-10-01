package dsl

import "github.com/BCBP-SOLUTIONS-FZC-LLC/workflow-models/pkg/enums"

// StageDef is one compiled user/service task stage within a DepartmentDef.
type StageDef struct {
	Type             string            `json:"type"`
	Activity         string            `json:"activity"`
	NodeID           string            `json:"node_id,omitempty"`
	Name             string            `json:"name,omitempty"`
	Role             string            `json:"role"`
	DefaultAssignees []string          `json:"default_assignees,omitempty"`
	DueDate          string            `json:"due_date,omitempty"`
	FollowUpDate     string            `json:"follow_up_date,omitempty"`
	BoundaryTimer    *BoundaryTimer    `json:"boundary_timer,omitempty"`
	BoundaryMessage  *MessagePath      `json:"boundary_message,omitempty"`
	EngineNote       string            `json:"engine_note,omitempty"`
	Extras           map[string]string `json:"extras,omitempty"`
	IsZeebeUserTask  bool              `json:"is_zeebe_user_task,omitempty"`
	ConnectorType    string            `json:"connector_type,omitempty"`
	IOMapping        *IOMapping        `json:"io_mapping,omitempty"`
}

// BoundaryTimer is a timer-boundary-event attached to a StageDef. Like every
// boundary, it either moves to TargetDept or, with Terminates, ends its path
// at an end event; ExpandCalls refuses one that does both or neither.
type BoundaryTimer struct {
	Duration     string `json:"duration"`
	Interrupting bool   `json:"interrupting"`
	TargetDept   string `json:"target_dept,omitempty"`
	Terminates   bool   `json:"terminates,omitempty"`
}

// NodeKey names a stage in both services: its department and its NodeID, or
// its Type when it has none. See workflow-models LLD §2.3.
func NodeKey(deptID string, s *StageDef) string {
	if s.NodeID != "" {
		return deptID + "/" + s.NodeID
	}
	return deptID + "/" + s.Type
}

// CreatesHumanTask reports whether the stage becomes a task a person
// completes, and so needs an assignee.
func (s *StageDef) CreatesHumanTask() bool {
	if s.ConnectorType != "" {
		return false
	}
	return s.Type != string(enums.StageTypeSendTask) && s.Type != string(enums.StageTypeReceiveTask)
}
