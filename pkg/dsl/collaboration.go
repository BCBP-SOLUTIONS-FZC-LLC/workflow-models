// Package dsl holds the compiled-plan DSL types shared between
// workflow-definition-service (producer) and the workflow execution
// service (consumer).
package dsl

// CurrentSchemaVersion is the DSL schema major version this module encodes.
// See workflow-models LLD §6.1.
const CurrentSchemaVersion = 1

// CompiledCollaboration is the root artifact of a published workflow:
// MainPlan names the plan that runs, the document's one runnable pool, and
// Plans holds it with the plan of every module it calls. Definition's BPMN
// compiler produces every field; Execution's workflow function consumes
// every field — the full contract, not a curated subset.
type CompiledCollaboration struct {
	MainPlan      string          `json:"main_plan"`
	Plans         []*CompiledPlan `json:"plans"`
	SchemaVersion int             `json:"schema_version"`
}
