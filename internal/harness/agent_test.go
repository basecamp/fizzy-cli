package harness

import "testing"

func TestRegisterAgent(t *testing.T) {
	t.Run("registers and finds agent", func(t *testing.T) {
		resetRegistry()
		defer resetRegistry()

		RegisterAgent(AgentInfo{
			Name:   "Test Agent",
			ID:     "test",
			Detect: func() bool { return true },
		})

		agent := FindAgent("test")
		if agent == nil {
			t.Fatal("expected to find registered agent")
			return
		}
		if agent.Name != "Test Agent" {
			t.Errorf("expected name 'Test Agent', got %q", agent.Name)
		}
	})

	t.Run("panics on empty ID", func(t *testing.T) {
		resetRegistry()
		defer resetRegistry()

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for empty ID")
			}
		}()
		RegisterAgent(AgentInfo{Name: "No ID"})
	})

	t.Run("panics on duplicate ID", func(t *testing.T) {
		resetRegistry()
		defer resetRegistry()

		RegisterAgent(AgentInfo{ID: "dup", Name: "First"})
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for duplicate ID")
			}
		}()
		RegisterAgent(AgentInfo{ID: "dup", Name: "Second"})
	})
}

func TestDetectedAgents(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	RegisterAgent(AgentInfo{
		ID:     "yes",
		Name:   "Detected",
		Detect: func() bool { return true },
	})
	RegisterAgent(AgentInfo{
		ID:     "no",
		Name:   "Not Detected",
		Detect: func() bool { return false },
	})

	detected := DetectedAgents()
	if len(detected) != 1 {
		t.Fatalf("expected 1 detected agent, got %d", len(detected))
	}
	if detected[0].ID != "yes" {
		t.Errorf("expected detected agent 'yes', got %q", detected[0].ID)
	}
}

func TestAllAgents(t *testing.T) {
	resetRegistry()
	defer resetRegistry()

	RegisterAgent(AgentInfo{ID: "a", Name: "A"})
	RegisterAgent(AgentInfo{ID: "b", Name: "B"})

	all := AllAgents()
	if len(all) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(all))
	}
}
