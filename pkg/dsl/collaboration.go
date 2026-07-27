// Package dsl holds the compiled-plan DSL types shared between
// workflow-definition-service (producer) and the future workflow execution
// service (consumer).
package dsl

// CompiledCollaboration is the root compiled-plan artifact: one top-level
// collaboration containing every compiled pool (CompiledPlan) and the
// inter-pool messages between them.
type CompiledCollaboration struct {
	MainPlan string          `json:"main_plan"`
	Plans    []*CompiledPlan `json:"plans"`
	Messages []MessageDef    `json:"messages"`
}

// MessageDef describes one inter-pool BPMN message: the sending and
// receiving pool (process) names.
type MessageDef struct {
	Name       string `json:"name"`
	SourcePlan string `json:"source_plan"` // process name of the sending participant
	TargetPlan string `json:"target_plan"` // process name of the receiving participant
}
