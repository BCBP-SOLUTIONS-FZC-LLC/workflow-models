package dsl_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/BCBP-SOLUTIONS-FZC-LLC/workflow-models/pkg/dsl"
)

// TestCompiledCollaboration_RoundTrip is the golden drift tripwire: every
// struct/field in pkg/dsl marshals to JSON and back to an identical value.
// A field that stops round-tripping means this package's shape has drifted
// from workflow-definition-service's real internal/core/domain/compiled_plan.go.
func TestCompiledCollaboration_RoundTrip(t *testing.T) {
	original := &dsl.CompiledCollaboration{
		MainPlan:      "Bid-No-Bid Review",
		SchemaVersion: dsl.CurrentSchemaVersion,
		Plans: []*dsl.CompiledPlan{
			{
				Name:      "Bid-No-Bid Review",
				TaskQueue: "wf-queue-default",
				Ignored:   false,
				Departments: []dsl.DepartmentDef{
					{
						ID:              "Bid-No-Bid Review/Tender Business",
						Label:           "Tender Business",
						IAMDepartmentID: "018e1f2a-0000-7000-8000-000000000001",
						Ignore:          false,
						Props:           map[string]string{"sla": "24h"},
						Stages: []dsl.StageDef{
							{
								Type:             "receive_task",
								Activity:         "RFQ received",
								NodeID:           "Activity_1qy77cx",
								Name:             "RFQ received",
								Role:             "tender-business",
								DefaultAssignees: []string{"018e1f2a-0000-7000-8000-000000000020"},
								DueDate:          "P3D",
								FollowUpDate:     "P1D",
								BoundaryTimer: &dsl.BoundaryTimer{
									Duration:     "72h",
									Interrupting: true,
									TargetDept:   "escalation",
								},
								BoundaryMessage: &dsl.MessagePath{
									MessageName:  "cancel",
									Interrupting: true,
									TargetDept:   "ops",
								},
								EngineNote:      "",
								Extras:          map[string]string{"message": "rfq"},
								IsZeebeUserTask: true,
							},
							{
								Type:          "connector",
								Activity:      "Fetch tender packet",
								NodeID:        "Activity_connector1",
								Role:          "tender-business",
								ConnectorType: "storage",
								IOMapping: &dsl.IOMapping{
									Inputs:  []dsl.IOVar{{Source: "=tenderId", Target: "key"}},
									Outputs: []dsl.IOVar{{Source: "=contentRef", Target: "packet_ref"}},
								},
							},
						},
					},
				},
				Execution: dsl.ExecutionPlan{
					Steps: []dsl.ExecutionStep{
						{Sequential: []string{"Bid-No-Bid Review/Tender Business"}},
						{
							Parallel: []dsl.ParallelBranch{
								{DeptID: "ops", Steps: []dsl.ExecutionStep{{Sequential: []string{"ops"}}}},
							},
						},
						{
							Exclusive: []dsl.ExclusiveBranch{
								{
									Target:              "ops",
									TargetStage:         "prep",
									TargetNodeID:        "Activity_prep",
									TargetName:          "Prep",
									ConditionExpression: "",
									Terminates:          false,
									RevertToDept:        "design",
									RevertToStage:       "review",
									RevertToNodeID:      "Activity_review",
									RevertToName:        "Review",
								},
							},
						},
						{
							SubWorkflow: &dsl.SubWorkflowStep{
								NodeID: "SubProcess_1",
								Name:   "Escalation",
								Plan:   dsl.ExecutionPlan{Steps: []dsl.ExecutionStep{{Sequential: []string{"escalation"}}}},
								ErrorPaths: []dsl.ErrorPath{
									{ErrorCode: "ERR_TIMEOUT", ShortCircuit: true, TargetDept: "ops"},
								},
								TimerPaths: []dsl.TimerPath{
									{Duration: "24h", Interrupting: false, TargetDept: "ops"},
								},
								MessagePaths: []dsl.MessagePath{
									{MessageName: "escalate", Interrupting: true, TargetDept: "ops"},
								},
							},
						},
						{
							CallPool: &dsl.CallPoolStep{Pool: "Issue RFQ"},
							IOMapping: &dsl.IOMapping{
								Inputs:  []dsl.IOVar{{Source: "=tenderId", Target: "tender_id"}},
								Outputs: []dsl.IOVar{{Source: "=result", Target: "outcome"}},
							},
							Extras:       map[string]string{"exec.priority": "high"},
							MessagePaths: []dsl.MessagePath{{MessageName: "notify", Interrupting: false, Terminates: true}},
						},
						{
							CallPlan: &dsl.CallPlanStep{
								NodeID:       "Activity_call1",
								Name:         "Prepare response",
								Plan:         "prepare_response@v3",
								Departments:  map[string]string{"Sender": "Bid-No-Bid Review/Tender Business"},
								Assignees:    map[string]string{"Task_prepare": "018e1f2a-0000-7000-8000-000000000021"},
								ErrorPaths:   []dsl.ErrorPath{{ErrorCode: "E_REJECT", ShortCircuit: true, TargetDept: "escalation"}, {Terminates: true}},
								TimerPaths:   []dsl.TimerPath{{Duration: "P5D", Interrupting: true, TargetDept: "escalation"}, {Duration: "P9D", Interrupting: true, Terminates: true}},
								MessagePaths: []dsl.MessagePath{{MessageName: "withdraw", Interrupting: true, TargetDept: "ops"}},
							},
						},
					},
				},
				VisualElements: []dsl.VisualElementDef{
					{Kind: "dataStoreReference", ID: "DataStoreReference_1", Name: "Archive"},
				},
			},
		},
		Messages: []dsl.MessageDef{
			{Name: "RFQ", SourcePlan: "Issue RFQ", TargetPlan: "Bid-No-Bid Review"},
		},
	}

	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got dsl.CompiledCollaboration
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !reflect.DeepEqual(original, &got) {
		t.Fatalf("round-trip mismatch:\n original=%+v\n got     =%+v", original, got)
	}
}
