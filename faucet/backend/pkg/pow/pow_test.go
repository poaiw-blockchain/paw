package pow

import (
	"testing"
	"time"
)

func TestNewProofOfWork(t *testing.T) {
	tests := []struct {
		name               string
		difficulty         int
		expectedDifficulty int
	}{
		{"Default difficulty", 4, 4},
		{"Low difficulty", 2, 2},
		{"High difficulty", 6, 6},
		{"Too low (capped)", 0, DefaultDifficulty},
		{"Too high (capped)", 10, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pow := NewProofOfWork(tt.difficulty)
			if pow.GetDifficulty() != tt.expectedDifficulty {
				t.Errorf("expected difficulty %d, got %d", tt.expectedDifficulty, pow.GetDifficulty())
			}
		})
	}
}

func TestGenerateChallenge(t *testing.T) {
	pow := NewProofOfWork(4)
	resource := "paw1testaddress"

	challenge := pow.GenerateChallenge(resource)

	if challenge.Resource != resource {
		t.Errorf("expected resource %s, got %s", resource, challenge.Resource)
	}

	if challenge.Difficulty != 4 {
		t.Errorf("expected difficulty 4, got %d", challenge.Difficulty)
	}

	now := time.Now().Unix()
	if challenge.Timestamp > now || challenge.Timestamp < now-5 {
		t.Errorf("challenge timestamp out of range: %d", challenge.Timestamp)
	}

	expectedExpiry := challenge.Timestamp + int64(ChallengeTTL.Seconds())
	if challenge.ExpiresAt != expectedExpiry {
		t.Errorf("expected expiry %d, got %d", expectedExpiry, challenge.ExpiresAt)
	}
}

func TestVerifySolution_ValidSolution(t *testing.T) {
	pow := NewProofOfWork(2) // Low difficulty for faster testing
	challenge := pow.GenerateChallenge("paw1testaddress")

	// Compute a valid solution
	solution, err := pow.ComputeSolution(challenge)
	if err != nil {
		t.Fatalf("failed to compute solution: %v", err)
	}

	// Verify the solution
	err = pow.VerifySolution(challenge, solution)
	if err != nil {
		t.Errorf("valid solution verification failed: %v", err)
	}
}

func TestVerifySolution_InvalidNonce(t *testing.T) {
	pow := NewProofOfWork(4)
	challenge := pow.GenerateChallenge("paw1testaddress")

	// Create a solution with wrong nonce
	solution := &Solution{
		Resource:  challenge.Resource,
		Timestamp: challenge.Timestamp,
		Nonce:     "invalid",
	}

	err := pow.VerifySolution(challenge, solution)
	if err == nil {
		t.Error("expected verification to fail for invalid nonce")
	}
}

func TestVerifySolution_MismatchedResource(t *testing.T) {
	pow := NewProofOfWork(2)
	challenge := pow.GenerateChallenge("paw1testaddress")

	solution := &Solution{
		Resource:  "paw1differentaddress",
		Timestamp: challenge.Timestamp,
		Nonce:     "12345",
	}

	err := pow.VerifySolution(challenge, solution)
	if err == nil {
		t.Error("expected verification to fail for mismatched resource")
	}
}

func TestVerifySolution_MismatchedTimestamp(t *testing.T) {
	pow := NewProofOfWork(2)
	challenge := pow.GenerateChallenge("paw1testaddress")

	solution := &Solution{
		Resource:  challenge.Resource,
		Timestamp: challenge.Timestamp + 100,
		Nonce:     "12345",
	}

	err := pow.VerifySolution(challenge, solution)
	if err == nil {
		t.Error("expected verification to fail for mismatched timestamp")
	}
}

func TestVerifySolution_ExpiredChallenge(t *testing.T) {
	pow := NewProofOfWork(2)

	// Create an expired challenge
	challenge := &Challenge{
		Resource:   "paw1testaddress",
		Timestamp:  time.Now().Add(-10 * time.Minute).Unix(),
		Difficulty: 2,
		ExpiresAt:  time.Now().Add(-5 * time.Minute).Unix(),
	}

	solution := &Solution{
		Resource:  challenge.Resource,
		Timestamp: challenge.Timestamp,
		Nonce:     "12345",
	}

	err := pow.VerifySolution(challenge, solution)
	if err == nil {
		t.Error("expected verification to fail for expired challenge")
	}
}

func TestVerifySolution_FutureTimestamp(t *testing.T) {
	pow := NewProofOfWork(2)

	// Create a challenge with future timestamp
	challenge := &Challenge{
		Resource:   "paw1testaddress",
		Timestamp:  time.Now().Add(2 * time.Minute).Unix(),
		Difficulty: 2,
		ExpiresAt:  time.Now().Add(7 * time.Minute).Unix(),
	}

	solution := &Solution{
		Resource:  challenge.Resource,
		Timestamp: challenge.Timestamp,
		Nonce:     "12345",
	}

	err := pow.VerifySolution(challenge, solution)
	if err == nil {
		t.Error("expected verification to fail for future timestamp")
	}
}

func TestVerifySolution_TooLongNonce(t *testing.T) {
	pow := NewProofOfWork(2)
	challenge := pow.GenerateChallenge("paw1testaddress")

	// Create a solution with excessively long nonce
	longNonce := string(make([]byte, MaxNonceLength+1))
	solution := &Solution{
		Resource:  challenge.Resource,
		Timestamp: challenge.Timestamp,
		Nonce:     longNonce,
	}

	err := pow.VerifySolution(challenge, solution)
	if err == nil {
		t.Error("expected verification to fail for too long nonce")
	}
}

func TestSetDifficulty(t *testing.T) {
	pow := NewProofOfWork(4)

	pow.SetDifficulty(6)
	if pow.GetDifficulty() != 6 {
		t.Errorf("expected difficulty 6, got %d", pow.GetDifficulty())
	}

	// Test capping
	pow.SetDifficulty(10)
	if pow.GetDifficulty() != 8 {
		t.Errorf("expected difficulty to be capped at 8, got %d", pow.GetDifficulty())
	}

	pow.SetDifficulty(0)
	if pow.GetDifficulty() != 1 {
		t.Errorf("expected difficulty to be at least 1, got %d", pow.GetDifficulty())
	}
}

func TestEstimateWorkTime(t *testing.T) {
	tests := []struct {
		difficulty int
		maxTime    time.Duration
	}{
		{1, 1 * time.Millisecond},
		{2, 100 * time.Millisecond},
		{3, 5 * time.Second},
		{4, 1 * time.Minute},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			estimated := EstimateWorkTime(tt.difficulty)
			// Just check it returns a reasonable value
			if estimated > 1*time.Hour {
				t.Errorf("estimated work time too high for difficulty %d: %v", tt.difficulty, estimated)
			}
		})
	}
}

func BenchmarkComputeSolution_Difficulty2(b *testing.B) {
	pow := NewProofOfWork(2)
	challenge := pow.GenerateChallenge("paw1testaddress")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pow.ComputeSolution(challenge)
	}
}

func BenchmarkComputeSolution_Difficulty3(b *testing.B) {
	pow := NewProofOfWork(3)
	challenge := pow.GenerateChallenge("paw1testaddress")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pow.ComputeSolution(challenge)
	}
}

func BenchmarkVerifySolution(b *testing.B) {
	pow := NewProofOfWork(2)
	challenge := pow.GenerateChallenge("paw1testaddress")
	solution, _ := pow.ComputeSolution(challenge)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pow.VerifySolution(challenge, solution)
	}
}
