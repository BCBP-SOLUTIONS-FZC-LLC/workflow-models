// Package dsl holds the compiled-plan DSL types shared between
// workflow-definition-service (producer) and the workflow execution
// service (consumer).
package dsl

// CurrentSchemaVersion is the DSL schema major version this module encodes.
// See workflow-models LLD §6.1.
const CurrentSchemaVersion = 1

// CompiledCollaboration is the root artifact of a compiled BPMN
// collaboration: MainPlan names the entry pool, Plans holds every compiled
// pool (including sub-processes), and Messages holds every inter-pool
// message flow. Definition's BPMN compiler produces every field; Execution's
// workflow function consumes every field — the full contract, not a
// curated subset.
type CompiledCollaboration struct {
	MainPlan      string          `json:"main_plan"`
	Plans         []*CompiledPlan `json:"plans"`
	Messages      []MessageDef    `json:"messages"`
	SchemaVersion int             `json:"schema_version"`
}

// MessageDef is one inter-pool BPMN message flow, connecting the sending
// participant's process (SourcePlan) to the receiving participant's process
// (TargetPlan).
type MessageDef struct {
	Name       string `json:"name"`
	SourcePlan string `json:"source_plan"` // process name of the sending participant
	TargetPlan string `json:"target_plan"` // process name of the receiving participant
}
