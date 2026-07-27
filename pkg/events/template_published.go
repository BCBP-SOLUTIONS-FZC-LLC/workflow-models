// Package events holds the shared event payload types published by
// workflow-definition-service and consumed by the future execution service.
package events

// TemplatePublishedPayload is the payload of the workflow.template.published
// event (EventTypeTemplatePublished), published when Definition Service
// publishes or promotes a workflow template version.
type TemplatePublishedPayload struct {
	WorkflowID            string  `json:"workflow_id"`
	WorkflowKey           string  `json:"workflow_key"`
	VersionID             string  `json:"version_id"`
	VersionNumber         int32   `json:"version_number"`
	ArtifactHash          string  `json:"artifact_hash"`
	PublishedBy           string  `json:"published_by"`
	PromotedFromVersionID *string `json:"promoted_from_version_id,omitempty"`
}
