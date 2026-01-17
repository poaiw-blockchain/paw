package circuits

import "testing"

func TestCircuitGetters(t *testing.T) {
	t.Run("compute circuit getters", func(t *testing.T) {
		var c ComputeCircuit
		if got := c.GetConstraintCount(); got == 0 {
			t.Fatalf("expected constraint count > 0")
		}
		if got := c.GetPublicInputCount(); got <= 0 {
			t.Fatalf("expected public inputs, got %d", got)
		}
		if got := c.GetCircuitName(); got == "" {
			t.Fatalf("expected circuit name")
		}
	})

	t.Run("escrow circuit getters", func(t *testing.T) {
		var c EscrowCircuit
		if got := c.GetConstraintCount(); got == 0 {
			t.Fatalf("expected constraint count > 0")
		}
		if got := c.GetPublicInputCount(); got <= 0 {
			t.Fatalf("expected public inputs, got %d", got)
		}
		if got := c.GetCircuitName(); got == "" {
			t.Fatalf("expected circuit name")
		}
	})

	t.Run("result circuit getters", func(t *testing.T) {
		var c ResultCircuit
		if got := c.GetConstraintCount(); got == 0 {
			t.Fatalf("expected constraint count > 0")
		}
		if got := c.GetPublicInputCount(); got <= 0 {
			t.Fatalf("expected public inputs, got %d", got)
		}
		if got := c.GetCircuitName(); got == "" {
			t.Fatalf("expected circuit name")
		}
	})
}
