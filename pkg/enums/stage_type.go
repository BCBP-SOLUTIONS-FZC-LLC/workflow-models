// Package enums holds the shared wire-level discriminator constants used by
// pkg/dsl and pkg/events.
package enums

// StageType values are the compiled-plan contract's stage-type discriminators
// — Definition writes them into StageDef.Type, Execution reads them. An
// unrecognized value is a valid forward-compat passthrough, not an exhaustive
// set; these five are the ones the compiler's own StageTypeHandler registry
// and message-task handlers currently emit.
type StageType string

// Recognized StageDef.Type values — see the StageType doc comment above for
// the forward-compat (non-exhaustive) policy.
const (
	StageTypePrep        StageType = "prep"
	StageTypeReview      StageType = "review"
	StageTypeApprove     StageType = "approve"
	StageTypeSendTask    StageType = "send_task"
	StageTypeReceiveTask StageType = "receive_task"
)

// EventTypeTemplatePublished is the wire event-type string for
// TemplatePublishedPayload, shared between Definition (producer) and
// Execution (consumer).
const EventTypeTemplatePublished = "workflow.template.published"
