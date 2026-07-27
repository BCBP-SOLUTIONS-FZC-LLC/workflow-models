package events_test

import (
	"encoding/json"
	"testing"

	"github.com/BCBP-SOLUTIONS-FZC-LLC/workflow-models/pkg/events"
)

// TestTemplatePublishedPayload_RoundTrip is the golden drift tripwire for the
// one shared event payload. Shape cross-checked against Definition Service's
// registered internal/eventschema/workflow_template_published.json: required
// fields workflow_id/workflow_key/version_id/version_number/artifact_hash/
// published_by, plus the optional nullable promoted_from_version_id.
func TestTemplatePublishedPayload_RoundTrip(t *testing.T) {
	promotedFrom := "018e1f2a-0000-7000-8000-000000000009"
	original := events.TemplatePublishedPayload{
		WorkflowID:            "018e1f2a-0000-7000-8000-000000000001",
		WorkflowKey:           "bid-no-bid-review",
		VersionID:             "018e1f2a-0000-7000-8000-000000000002",
		VersionNumber:         3,
		ArtifactHash:          "sha256:deadbeef",
		PublishedBy:           "018e1f2a-0000-7000-8000-000000000003",
		PromotedFromVersionID: &promotedFrom,
	}

	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var required map[string]any
	if err := json.Unmarshal(b, &required); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}
	for _, field := range []string{
		"workflow_id", "workflow_key", "version_id", "version_number", "artifact_hash", "published_by",
	} {
		if _, ok := required[field]; !ok {
			t.Errorf("required field %q missing from marshaled payload", field)
		}
	}

	var got events.TemplatePublishedPayload
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.WorkflowID != original.WorkflowID || got.WorkflowKey != original.WorkflowKey ||
		got.VersionID != original.VersionID || got.VersionNumber != original.VersionNumber ||
		got.ArtifactHash != original.ArtifactHash || got.PublishedBy != original.PublishedBy {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", got, original)
	}
	if got.PromotedFromVersionID == nil || *got.PromotedFromVersionID != promotedFrom {
		t.Fatalf("PromotedFromVersionID round-trip mismatch: got %v, want %q", got.PromotedFromVersionID, promotedFrom)
	}

	// Nil optional field must also round-trip cleanly (omitempty).
	noPromotion := events.TemplatePublishedPayload{
		WorkflowID:    "018e1f2a-0000-7000-8000-000000000001",
		WorkflowKey:   "bid-no-bid-review",
		VersionID:     "018e1f2a-0000-7000-8000-000000000002",
		VersionNumber: 1,
		ArtifactHash:  "sha256:deadbeef",
		PublishedBy:   "018e1f2a-0000-7000-8000-000000000003",
	}
	b2, err := json.Marshal(noPromotion)
	if err != nil {
		t.Fatalf("marshal (no promotion): %v", err)
	}
	var got2 events.TemplatePublishedPayload
	if err := json.Unmarshal(b2, &got2); err != nil {
		t.Fatalf("unmarshal (no promotion): %v", err)
	}
	if got2.PromotedFromVersionID != nil {
		t.Fatalf("expected nil PromotedFromVersionID, got %v", *got2.PromotedFromVersionID)
	}
}
