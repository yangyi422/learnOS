package ai

import "testing"

func TestParseAndValidateCurriculumDraftRejectsUnknownFields(t *testing.T) {
	_, err := ParseAndValidateCurriculumDraft(`{"summary":"ok","new_units":[],"new_lessons":[],"new_relations":[],"blueprint_mappings":[],"course":"should not be accepted"}`)
	if err == nil {
		t.Fatal("expected strict curriculum draft validation to reject unknown fields")
	}
}

func TestParseAndValidateCurriculumDraftNormalizesMissingArrays(t *testing.T) {
	result, err := ParseAndValidateCurriculumDraft(`{"summary":"补充课程骨架"}`)
	if err != nil {
		t.Fatalf("validate curriculum draft: %v", err)
	}
	if result.Summary != "补充课程骨架" || result.ChangeSetJSON == "" {
		t.Fatalf("unexpected normalized result: %+v", result)
	}
}
