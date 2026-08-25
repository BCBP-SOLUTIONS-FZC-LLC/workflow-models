// Package dsl holds the compiled-plan DSL types shared between
// workflow-definition-service (producer) and the workflow execution
// service (consumer).
package dsl

// CurrentSchemaVersion is the DSL schema major version this module encodes.
// See workflow_models_lib LLD §6.1.
const CurrentSchemaVersion = 1

type CompiledCollaboration struct {
	MainPlan      string          `json:"main_plan"`
	Plans         []*CompiledPlan `json:"plans"`
	Messages      []MessageDef    `json:"messages"`
	SchemaVersion int             `json:"schema_version"`
}

type MessageDef struct {
	Name       string `json:"name"`
	SourcePlan string `json:"source_plan"` // process name of the sending participant
	TargetPlan string `json:"target_plan"` // process name of the receiving participant
}
