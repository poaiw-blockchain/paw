package circuits

import (
	"hash"
	"math/big"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	mimcbn254 "github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark/std/algebra/native/twistededwards"
	"github.com/consensys/gnark/test"
)

func TestComputeCircuitValidAssignment(t *testing.T) {
	requestID := uint64(42)
	resultSize := uint64(0)
	chunkCount := uint64(0)
	exitCode := uint64(0)
	execTime := uint64(100)

	var resultBytes [32]uint64
	resultBytes[0] = 123
	var chunkValues [64]uint64
	chunkValues[0] = 777

	resourceMetrics := []uint64{1000, 2048, 4096, 2048, 1024, 512}
	providerNonce := uint64(77)
	providerSalt := uint64(88)

	resultCommitment := computeResultCommitmentValue(resultBytes, resultSize)
	computationCommitment := computeComputationHash(chunkValues, chunkCount, exitCode, execTime)
	_ = computationCommitment // ensures final hash path exercised; constraint only checks non-zero
	resourceCommitment := computeResourceCommitment(resourceMetrics)
	providerCommitment := computeProviderCommitment(providerNonce, providerSalt, requestID)

	assignment := &ComputeCircuit{
		RequestID:            requestID,
		ResultCommitment:     resultCommitment,
		ProviderCommitment:   providerCommitment,
		ResourceCommitment:   resourceCommitment,
		ChunkCount:           chunkCount,
		ResultSize:           resultSize,
		ExecutionTimestamp:   execTime,
		ExitCode:             exitCode,
		CpuCyclesUsed:        resourceMetrics[0],
		MemoryBytesUsed:      resourceMetrics[1],
		DiskBytesRead:        resourceMetrics[2],
		DiskBytesWritten:     resourceMetrics[3],
		NetworkBytesReceived: resourceMetrics[4],
		NetworkBytesSent:     resourceMetrics[5],
		ProviderNonce:        providerNonce,
		ProviderSalt:         providerSalt,
	}

	for i := range resultBytes {
		assignment.ResultData[i] = resultBytes[i]
	}
	for i := range chunkValues {
		assignment.ComputationChunks[i] = chunkValues[i]
	}

	assert := test.NewAssert(t)
	assert.SolvingSucceeded(new(ComputeCircuit), assignment, test.WithCurves(ecc.BN254))
}

func TestComputeCircuitRejectsBadResourceCommitment(t *testing.T) {
	requestID := uint64(1)
	resultSize := uint64(0)
	chunkCount := uint64(0)

	var resultBytes [32]uint64
	resultBytes[0] = 55
	var chunkValues [64]uint64
	chunkValues[0] = 66
	resourceMetrics := []uint64{10, 20, 30, 40, 50, 60}

	assignment := &ComputeCircuit{
		RequestID:            requestID,
		ResultCommitment:     computeResultCommitmentValue(resultBytes, resultSize),
		ProviderCommitment:   computeProviderCommitment(1, 2, requestID),
		ResourceCommitment:   new(big.Int).SetUint64(999), // incorrect commitment
		ChunkCount:           chunkCount,
		ResultSize:           resultSize,
		ExecutionTimestamp:   10,
		ExitCode:             0,
		CpuCyclesUsed:        resourceMetrics[0],
		MemoryBytesUsed:      resourceMetrics[1],
		DiskBytesRead:        resourceMetrics[2],
		DiskBytesWritten:     resourceMetrics[3],
		NetworkBytesReceived: resourceMetrics[4],
		NetworkBytesSent:     resourceMetrics[5],
		ProviderNonce:        1,
		ProviderSalt:         2,
	}

	for i := range resultBytes {
		assignment.ResultData[i] = resultBytes[i]
	}
	for i := range chunkValues {
		assignment.ComputationChunks[i] = chunkValues[i]
	}

	assert := test.NewAssert(t)
	assert.SolvingFailed(new(ComputeCircuit), assignment, test.WithCurves(ecc.BN254))
}

func TestEscrowCircuitReleaseAndMisbehavior(t *testing.T) {
	baseEscrow := buildEscrowAssignment(1, true)
	assert := test.NewAssert(t)
	assert.SolvingSucceeded(new(EscrowCircuit), baseEscrow, test.WithCurves(ecc.BN254))

	bad := buildEscrowAssignment(1, false)
	assert.SolvingFailed(new(EscrowCircuit), bad, test.WithCurves(ecc.BN254))
}

func TestResultCircuitValidAndInvalidRoots(t *testing.T) {
	valid := buildResultAssignment()
	assert := test.NewAssert(t)
	assert.SolvingSucceeded(new(ResultCircuit), valid, test.WithCurves(ecc.BN254))

	invalid := buildResultAssignment()
	if root, ok := invalid.ResultRootHash.(*big.Int); ok {
		invalid.ResultRootHash = new(big.Int).Add(root, big.NewInt(1))
	}
	assert.SolvingFailed(new(ResultCircuit), invalid, test.WithCurves(ecc.BN254))
}

func buildEscrowAssignment(requestID uint64, honestProvider bool) *EscrowCircuit {
	var requesterAddr, providerAddr [20]uint64
	for i := 0; i < 20; i++ {
		requesterAddr[i] = uint64(i + 1)
		providerAddr[i] = uint64(i + 21)
	}
	requestNonce := uint64(5)
	requesterCommitment := hashRequester(requesterAddr, requestNonce)
	providerCommitment := hashProvider(providerAddr, requestID)

	resultHash := uint64(999)
	startTimestamp := uint64(10)
	deadlineTimestamp := uint64(100)
	endTimestamp := deadlineTimestamp
	completionCommitment := computeCompletionCommitment(resultHash, 1, 1, endTimestamp)

	actualCost := uint64(1)
	escrowAmount := actualCost
	providerMisbehavior := uint64(0)
	if !honestProvider {
		providerMisbehavior = 1
	}

	assign := &EscrowCircuit{
		RequestID:            requestID,
		EscrowAmount:         escrowAmount,
		RequesterCommitment:  requesterCommitment,
		ProviderCommitment:   providerCommitment,
		CompletionCommitment: completionCommitment,
		RequestNonce:         requestNonce,
		ResultHash:           resultHash,
		ExecutionSuccess:     1,
		VerificationPassed:   1,
		EstimatedCost:        actualCost + 100,
		ActualCost:           actualCost,
		ProviderMisbehavior:  providerMisbehavior,
		RequesterCancelled:   0,
		StartTimestamp:       startTimestamp,
		EndTimestamp:         endTimestamp,
		DeadlineTimestamp:    deadlineTimestamp,
	}

	for i := range requesterAddr {
		assign.RequesterAddress[i] = requesterAddr[i]
		assign.ProviderAddress[i] = providerAddr[i]
	}

	return assign
}

func buildResultAssignment() *ResultCircuit {
	requestID := uint64(7)
	leafIndex := uint64(0)
	var resultLeaves, inputLeaves [16]uint64
	resultLeaves[0] = 1234
	inputLeaves[0] = 4321

	var pathValues [4]uint64
	for i := range pathValues {
		pathValues[i] = uint64(i + 10)
	}

	resultRoot := computeMerkleRoot(resultLeaves, pathValues, leafIndex)
	inputRoot := computeMerkleRoot(inputLeaves, pathValues, leafIndex)
	programHash := big.NewInt(777)

	var traceSteps [32]uint64
	traceSteps[0] = 55
	traceSteps[1] = 77
	traceStepCount := uint64(1)

	var transitions [16]uint64
	for i := range transitions {
		transitions[i] = uint64(i + 1)
	}

	assign := &ResultCircuit{
		RequestID:       requestID,
		ResultRootHash:  resultRoot,
		InputRootHash:   inputRoot,
		ProgramHash:     programHash,
		ResultLeafIndex: leafIndex,
		InputLeafIndex:  leafIndex,
		TraceStepCount:  traceStepCount,
		RandomSeed:      99,
	}

	for i := range resultLeaves {
		assign.ResultLeaves[i] = resultLeaves[i]
		assign.InputLeaves[i] = inputLeaves[i]
	}
	for level := 0; level < 4; level++ {
		for sibling := 0; sibling < 16; sibling++ {
			value := uint64(0)
			if sibling == 0 {
				value = pathValues[level]
			}
			assign.ResultMerklePath[level][sibling] = value
			assign.InputMerklePath[level][sibling] = value
		}
	}
	for i := range traceSteps {
		assign.TraceSteps[i] = traceSteps[i]
	}
	for i := range transitions {
		assign.StateTransitions[i] = transitions[i]
	}

	assign.ProviderPublicKey.A = twistededwards.Point{
		X: 5,
		Y: 7,
	}
	assign.ResultSignature.R = twistededwards.Point{
		X: 9,
		Y: 11,
	}
	assign.ResultSignature.S = 13

	return assign
}

func computeResultCommitmentValue(resultBytes [32]uint64, resultSize uint64) *big.Int {
	h := mimcbn254.NewMiMC()
	for i := 0; i < len(resultBytes); i++ {
		val := uint64(0)
		if uint64(i) == resultSize {
			val = resultBytes[i]
		}
		writeUint64(h, val)
	}
	writeUint64(h, resultSize)
	return sumHash(h)
}

func computeComputationHash(chunks [64]uint64, chunkCount, exitCode, timestamp uint64) *big.Int {
	h := mimcbn254.NewMiMC()
	for i := 0; i < len(chunks); i++ {
		val := uint64(0)
		if uint64(i) == chunkCount {
			val = chunks[i]
		}
		writeUint64(h, val)
	}
	writeUint64(h, chunkCount)
	writeUint64(h, exitCode)
	writeUint64(h, timestamp)
	return sumHash(h)
}

func computeResourceCommitment(metrics []uint64) *big.Int {
	h := mimcbn254.NewMiMC()
	for _, v := range metrics {
		writeUint64(h, v)
	}
	return sumHash(h)
}

func computeProviderCommitment(nonce, salt, requestID uint64) *big.Int {
	h := mimcbn254.NewMiMC()
	writeUint64(h, nonce)
	writeUint64(h, salt)
	writeUint64(h, requestID)
	return sumHash(h)
}

func hashRequester(addr [20]uint64, nonce uint64) *big.Int {
	h := mimcbn254.NewMiMC()
	for _, v := range addr {
		writeUint64(h, v)
	}
	writeUint64(h, nonce)
	return sumHash(h)
}

func hashProvider(addr [20]uint64, requestID uint64) *big.Int {
	h := mimcbn254.NewMiMC()
	for _, v := range addr {
		writeUint64(h, v)
	}
	writeUint64(h, requestID)
	return sumHash(h)
}

func computeCompletionCommitment(resultHash, success, verified, end uint64) *big.Int {
	h := mimcbn254.NewMiMC()
	writeUint64(h, resultHash)
	writeUint64(h, success)
	writeUint64(h, verified)
	writeUint64(h, end)
	return sumHash(h)
}

func computeMerkleRoot(leaves [16]uint64, pathVals [4]uint64, index uint64) *big.Int {
	h := mimcbn254.NewMiMC()
	for _, leaf := range leaves {
		writeUint64(h, leaf)
	}
	current := sumHash(h)

	for level := 0; level < 4; level++ {
		h.Reset()
		if ((index >> level) & 1) == 1 {
			writeUint64(h, pathVals[level])
			writeBigInt(h, current)
		} else {
			writeBigInt(h, current)
			writeUint64(h, pathVals[level])
		}
		current = sumHash(h)
	}
	return current
}

func writeUint64(h hash.Hash, v uint64) {
	var el fr.Element
	el.SetUint64(v)
	bytes := el.Bytes()
	h.Write(bytes[:])
}

func writeBigInt(h hash.Hash, v *big.Int) {
	var el fr.Element
	el.SetBigInt(v)
	bytes := el.Bytes()
	h.Write(bytes[:])
}

func sumHash(h hash.Hash) *big.Int {
	return new(big.Int).SetBytes(h.Sum(nil))
}
