package types

import (
	"context"
	"errors"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// mockComputeHooks implements ComputeHooks for testing
type mockComputeHooks struct {
	jobCompletedCalled       bool
	jobFailedCalled          bool
	providerRegisteredCalled bool
	providerSlashedCalled    bool
	circuitBreakerCalled     bool
	returnError              error
}

func (m *mockComputeHooks) AfterJobCompleted(ctx context.Context, requestID uint64, provider sdk.AccAddress, result []byte) error {
	m.jobCompletedCalled = true
	return m.returnError
}

func (m *mockComputeHooks) AfterJobFailed(ctx context.Context, requestID uint64, reason string) error {
	m.jobFailedCalled = true
	return m.returnError
}

func (m *mockComputeHooks) AfterProviderRegistered(ctx context.Context, provider sdk.AccAddress, stake sdkmath.Int) error {
	m.providerRegisteredCalled = true
	return m.returnError
}

func (m *mockComputeHooks) AfterProviderSlashed(ctx context.Context, provider sdk.AccAddress, slashAmount sdkmath.Int, reason string) error {
	m.providerSlashedCalled = true
	return m.returnError
}

func (m *mockComputeHooks) OnCircuitBreakerTriggered(ctx context.Context, reason string) error {
	m.circuitBreakerCalled = true
	return m.returnError
}

func TestNewMultiComputeHooks(t *testing.T) {
	hook1 := &mockComputeHooks{}
	hook2 := &mockComputeHooks{}

	multiHooks := NewMultiComputeHooks(hook1, hook2)

	if len(multiHooks) != 2 {
		t.Errorf("NewMultiComputeHooks() created %d hooks, want 2", len(multiHooks))
	}
}

func TestNewMultiComputeHooksEmpty(t *testing.T) {
	multiHooks := NewMultiComputeHooks()

	if len(multiHooks) != 0 {
		t.Errorf("NewMultiComputeHooks() created %d hooks, want 0", len(multiHooks))
	}
}

func TestMultiComputeHooks_AfterJobCompleted(t *testing.T) {
	tests := []struct {
		name        string
		hooks       []ComputeHooks
		expectError bool
	}{
		{
			name:        "empty hooks",
			hooks:       []ComputeHooks{},
			expectError: false,
		},
		{
			name:        "nil hook in slice",
			hooks:       []ComputeHooks{nil},
			expectError: false,
		},
		{
			name:        "single hook success",
			hooks:       []ComputeHooks{&mockComputeHooks{}},
			expectError: false,
		},
		{
			name:        "multiple hooks success",
			hooks:       []ComputeHooks{&mockComputeHooks{}, &mockComputeHooks{}},
			expectError: false,
		},
		{
			name:        "hook returns error",
			hooks:       []ComputeHooks{&mockComputeHooks{returnError: errors.New("test error")}},
			expectError: true,
		},
		{
			name: "error stops propagation",
			hooks: []ComputeHooks{
				&mockComputeHooks{returnError: errors.New("test error")},
				&mockComputeHooks{}, // This should not be called
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiHooks := MultiComputeHooks(tt.hooks)
			err := multiHooks.AfterJobCompleted(context.Background(), 1, sdk.AccAddress{}, []byte("result"))

			if (err != nil) != tt.expectError {
				t.Errorf("AfterJobCompleted() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestMultiComputeHooks_AfterJobFailed(t *testing.T) {
	tests := []struct {
		name        string
		hooks       []ComputeHooks
		expectError bool
	}{
		{
			name:        "empty hooks",
			hooks:       []ComputeHooks{},
			expectError: false,
		},
		{
			name:        "nil hook in slice",
			hooks:       []ComputeHooks{nil},
			expectError: false,
		},
		{
			name:        "single hook success",
			hooks:       []ComputeHooks{&mockComputeHooks{}},
			expectError: false,
		},
		{
			name:        "hook returns error",
			hooks:       []ComputeHooks{&mockComputeHooks{returnError: errors.New("test error")}},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiHooks := MultiComputeHooks(tt.hooks)
			err := multiHooks.AfterJobFailed(context.Background(), 1, "test reason")

			if (err != nil) != tt.expectError {
				t.Errorf("AfterJobFailed() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestMultiComputeHooks_AfterProviderRegistered(t *testing.T) {
	tests := []struct {
		name        string
		hooks       []ComputeHooks
		expectError bool
	}{
		{
			name:        "empty hooks",
			hooks:       []ComputeHooks{},
			expectError: false,
		},
		{
			name:        "nil hook in slice",
			hooks:       []ComputeHooks{nil},
			expectError: false,
		},
		{
			name:        "single hook success",
			hooks:       []ComputeHooks{&mockComputeHooks{}},
			expectError: false,
		},
		{
			name:        "hook returns error",
			hooks:       []ComputeHooks{&mockComputeHooks{returnError: errors.New("test error")}},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiHooks := MultiComputeHooks(tt.hooks)
			err := multiHooks.AfterProviderRegistered(context.Background(), sdk.AccAddress{}, sdkmath.NewInt(1000000))

			if (err != nil) != tt.expectError {
				t.Errorf("AfterProviderRegistered() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestMultiComputeHooks_AfterProviderSlashed(t *testing.T) {
	tests := []struct {
		name        string
		hooks       []ComputeHooks
		expectError bool
	}{
		{
			name:        "empty hooks",
			hooks:       []ComputeHooks{},
			expectError: false,
		},
		{
			name:        "nil hook in slice",
			hooks:       []ComputeHooks{nil},
			expectError: false,
		},
		{
			name:        "single hook success",
			hooks:       []ComputeHooks{&mockComputeHooks{}},
			expectError: false,
		},
		{
			name:        "hook returns error",
			hooks:       []ComputeHooks{&mockComputeHooks{returnError: errors.New("test error")}},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiHooks := MultiComputeHooks(tt.hooks)
			err := multiHooks.AfterProviderSlashed(context.Background(), sdk.AccAddress{}, sdkmath.NewInt(100000), "misbehavior")

			if (err != nil) != tt.expectError {
				t.Errorf("AfterProviderSlashed() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestMultiComputeHooks_OnCircuitBreakerTriggered(t *testing.T) {
	tests := []struct {
		name        string
		hooks       []ComputeHooks
		expectError bool
	}{
		{
			name:        "empty hooks",
			hooks:       []ComputeHooks{},
			expectError: false,
		},
		{
			name:        "nil hook in slice",
			hooks:       []ComputeHooks{nil},
			expectError: false,
		},
		{
			name:        "single hook success",
			hooks:       []ComputeHooks{&mockComputeHooks{}},
			expectError: false,
		},
		{
			name:        "hook returns error",
			hooks:       []ComputeHooks{&mockComputeHooks{returnError: errors.New("test error")}},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiHooks := MultiComputeHooks(tt.hooks)
			err := multiHooks.OnCircuitBreakerTriggered(context.Background(), "anomaly detected")

			if (err != nil) != tt.expectError {
				t.Errorf("OnCircuitBreakerTriggered() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestMultiComputeHooks_AllHooksCalled(t *testing.T) {
	hook1 := &mockComputeHooks{}
	hook2 := &mockComputeHooks{}
	multiHooks := NewMultiComputeHooks(hook1, hook2)

	ctx := context.Background()
	addr := sdk.AccAddress{}
	stake := sdkmath.NewInt(1000000)

	// Test AfterJobCompleted
	_ = multiHooks.AfterJobCompleted(ctx, 1, addr, []byte("result"))
	if !hook1.jobCompletedCalled || !hook2.jobCompletedCalled {
		t.Error("AfterJobCompleted should call all hooks")
	}

	// Test AfterJobFailed
	_ = multiHooks.AfterJobFailed(ctx, 1, "reason")
	if !hook1.jobFailedCalled || !hook2.jobFailedCalled {
		t.Error("AfterJobFailed should call all hooks")
	}

	// Test AfterProviderRegistered
	_ = multiHooks.AfterProviderRegistered(ctx, addr, stake)
	if !hook1.providerRegisteredCalled || !hook2.providerRegisteredCalled {
		t.Error("AfterProviderRegistered should call all hooks")
	}

	// Test AfterProviderSlashed
	_ = multiHooks.AfterProviderSlashed(ctx, addr, stake, "reason")
	if !hook1.providerSlashedCalled || !hook2.providerSlashedCalled {
		t.Error("AfterProviderSlashed should call all hooks")
	}

	// Test OnCircuitBreakerTriggered
	_ = multiHooks.OnCircuitBreakerTriggered(ctx, "reason")
	if !hook1.circuitBreakerCalled || !hook2.circuitBreakerCalled {
		t.Error("OnCircuitBreakerTriggered should call all hooks")
	}
}

func TestMultiComputeHooks_ErrorStopsPropagation(t *testing.T) {
	hook1 := &mockComputeHooks{returnError: errors.New("stop here")}
	hook2 := &mockComputeHooks{}
	multiHooks := NewMultiComputeHooks(hook1, hook2)

	ctx := context.Background()

	// Test AfterJobCompleted
	_ = multiHooks.AfterJobCompleted(ctx, 1, sdk.AccAddress{}, []byte("result"))
	if !hook1.jobCompletedCalled {
		t.Error("First hook should be called")
	}
	if hook2.jobCompletedCalled {
		t.Error("Second hook should not be called after error")
	}
}

func TestMultiComputeHooks_WithNilHooks(t *testing.T) {
	hook := &mockComputeHooks{}
	multiHooks := NewMultiComputeHooks(nil, hook, nil)

	ctx := context.Background()

	// Should not panic and should call the valid hook
	err := multiHooks.AfterJobCompleted(ctx, 1, sdk.AccAddress{}, []byte("result"))
	if err != nil {
		t.Errorf("AfterJobCompleted with nil hooks should not error: %v", err)
	}
	if !hook.jobCompletedCalled {
		t.Error("Valid hook should be called even with nil hooks in slice")
	}
}
