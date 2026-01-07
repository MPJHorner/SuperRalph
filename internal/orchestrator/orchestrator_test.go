package orchestrator

import (
	"encoding/json"
	"testing"
)

func TestResponseParsing(t *testing.T) {
	jsonStr := `{
		"thinking": "I should ask what they're building",
		"action": "ask_user",
		"action_params": {
			"question": "What are you building?"
		},
		"message": "Let's start by understanding your project.",
		"state": {"phase": "gathering"}
	}`

	var response Response
	err := json.Unmarshal([]byte(jsonStr), &response)
	if err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Action != ActionAskUser {
		t.Errorf("expected action %s, got %s", ActionAskUser, response.Action)
	}
	if response.ActionParams.Question != "What are you building?" {
		t.Errorf("unexpected question: %s", response.ActionParams.Question)
	}
	if response.Thinking != "I should ask what they're building" {
		t.Errorf("unexpected thinking: %s", response.Thinking)
	}
}

func TestActionTypes(t *testing.T) {
	tests := []struct {
		action Action
		valid  bool
	}{
		{ActionAskUser, true},
		{ActionReadFiles, true},
		{ActionWriteFile, true},
		{ActionRunCommand, true},
		{ActionDone, true},
		{Action("invalid"), false},
	}

	validActions := map[Action]bool{
		ActionAskUser:    true,
		ActionReadFiles:  true,
		ActionWriteFile:  true,
		ActionRunCommand: true,
		ActionDone:       true,
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			_, exists := validActions[tt.action]
			if exists != tt.valid {
				t.Errorf("action %s: expected valid=%v, got valid=%v", tt.action, tt.valid, exists)
			}
		})
	}
}

func TestSessionSerialization(t *testing.T) {
	session := &Session{
		ID:      "test-123",
		Mode:    "plan",
		WorkDir: "/tmp/test",
		Messages: []Message{
			{Role: "system", Content: "You are a helpful assistant"},
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
		},
		State: &PlanState{
			Phase: "gathering",
			DraftPRD: &DraftPRD{
				Name: "Test Project",
			},
		},
	}

	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("failed to marshal session: %v", err)
	}

	var restored Session
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("failed to unmarshal session: %v", err)
	}

	if restored.ID != session.ID {
		t.Errorf("expected ID %s, got %s", session.ID, restored.ID)
	}
	if len(restored.Messages) != len(session.Messages) {
		t.Errorf("expected %d messages, got %d", len(session.Messages), len(restored.Messages))
	}
}

func TestOrchestratorNew(t *testing.T) {
	orch := New("/tmp/test")
	if orch == nil {
		t.Fatal("expected non-nil orchestrator")
	}
	if orch.workDir != "/tmp/test" {
		t.Errorf("expected workDir /tmp/test, got %s", orch.workDir)
	}
	if orch.session == nil {
		t.Fatal("expected non-nil session")
	}
	if orch.session.ID == "" {
		t.Error("expected non-empty session ID")
	}
}

func TestOrchestratorSetDebug(t *testing.T) {
	orch := New("/tmp/test")
	orch.SetDebug(true)
	if !orch.debug {
		t.Error("expected debug to be true")
	}
}
