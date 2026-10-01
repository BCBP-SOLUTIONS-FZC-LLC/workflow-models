package dsl_test

import (
	"encoding/json"
	"testing"

	"github.com/BCBP-SOLUTIONS-FZC-LLC/workflow-models/pkg/dsl"
	"github.com/BCBP-SOLUTIONS-FZC-LLC/workflow-models/pkg/enums"
)

func TestNodeKey(t *testing.T) {
	for _, tc := range []struct {
		name, dept string
		stage      dsl.StageDef
		want       string
	}{
		{"by node id", "Lane_Eng", dsl.StageDef{Type: "prep", NodeID: "Task_Prep"}, "Lane_Eng/Task_Prep"},
		{"by type without a node id", "Lane_Eng", dsl.StageDef{Type: "review"}, "Lane_Eng/review"},
		{"in a called module", "CA_Eng" + dsl.CallScopeSeparator + "Process_Review", dsl.StageDef{Type: "review", NodeID: "Review_Task"}, "CA_Eng::Process_Review/Review_Task"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := dsl.NodeKey(tc.dept, &tc.stage); got != tc.want {
				t.Errorf("NodeKey = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStageDef_CreatesHumanTask(t *testing.T) {
	for _, tc := range []struct {
		name  string
		stage dsl.StageDef
		want  bool
	}{
		{"prep", dsl.StageDef{Type: string(enums.StageTypePrep)}, true},
		{"review", dsl.StageDef{Type: string(enums.StageTypeReview)}, true},
		{"approve", dsl.StageDef{Type: string(enums.StageTypeApprove)}, true},
		{"an unknown type", dsl.StageDef{Type: "legal_signoff"}, true},
		{"send task", dsl.StageDef{Type: string(enums.StageTypeSendTask)}, false},
		{"receive task", dsl.StageDef{Type: string(enums.StageTypeReceiveTask)}, false},
		{"connector", dsl.StageDef{Type: string(enums.StageTypeConnector), ConnectorType: "send-email"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.stage.CreatesHumanTask(); got != tc.want {
				t.Errorf("CreatesHumanTask = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStageDef_NameRoundTrips(t *testing.T) {
	raw, err := json.Marshal(dsl.StageDef{Type: "prep", NodeID: "Task_Prep", Name: "Prepare the response"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got dsl.StageDef
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Name != "Prepare the response" {
		t.Errorf("Name = %q after a round trip of %s", got.Name, raw)
	}
}
