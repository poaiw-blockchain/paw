package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestJobResultPacketDataValidateBasic(t *testing.T) {
	valid := types.JobResultPacketData{
		Type:      types.JobResultType,
		Nonce:     1,
		Timestamp: 1,
		JobID:     "job-1",
		Provider:  "prov-1",
		Result: types.JobResult{
			ResultData: []byte{0x1},
			ResultHash: "abc",
		},
	}

	tt := []struct {
		name    string
		mutate  func(p *types.JobResultPacketData)
		wantErr string
	}{
		{"bad type", func(p *types.JobResultPacketData) { p.Type = "bad" }, "invalid packet type"},
		{"zero nonce", func(p *types.JobResultPacketData) { p.Nonce = 0 }, "nonce"},
		{"zero timestamp", func(p *types.JobResultPacketData) { p.Timestamp = 0 }, "timestamp"},
		{"empty job id", func(p *types.JobResultPacketData) { p.JobID = "" }, "job ID"},
		{"empty result data", func(p *types.JobResultPacketData) { p.Result.ResultData = nil }, "result data"},
		{"empty result hash", func(p *types.JobResultPacketData) { p.Result.ResultHash = "" }, "result hash"},
		{"empty provider", func(p *types.JobResultPacketData) { p.Provider = "" }, "provider"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			p := valid
			tc.mutate(&p)
			err := p.ValidateBasic()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)
		})
	}

	require.NoError(t, valid.ValidateBasic())
}
