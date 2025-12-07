package ibcutil_test

import (
	"context"
	"testing"

	"github.com/paw-chain/paw/app/ibcutil"
	"github.com/stretchr/testify/require"
)

// mockChannelStore implements ibcutil.ChannelStore for testing.
type mockChannelStore struct {
	channels []ibcutil.AuthorizedChannel
	getErr   error
	setErr   error
}

func (m *mockChannelStore) GetAuthorizedChannels(ctx context.Context) ([]ibcutil.AuthorizedChannel, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.channels, nil
}

func (m *mockChannelStore) SetAuthorizedChannels(ctx context.Context, channels []ibcutil.AuthorizedChannel) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.channels = channels
	return nil
}

func TestAuthorizeChannel(t *testing.T) {
	tests := []struct {
		name           string
		initial        []ibcutil.AuthorizedChannel
		portID         string
		channelID      string
		expectedErr    bool
		expectedCount  int
		expectedResult []ibcutil.AuthorizedChannel
	}{
		{
			name:          "authorize new channel",
			initial:       []ibcutil.AuthorizedChannel{},
			portID:        "transfer",
			channelID:     "channel-0",
			expectedErr:   false,
			expectedCount: 1,
			expectedResult: []ibcutil.AuthorizedChannel{
				{PortId: "transfer", ChannelId: "channel-0"},
			},
		},
		{
			name: "authorize duplicate channel (idempotent)",
			initial: []ibcutil.AuthorizedChannel{
				{PortId: "transfer", ChannelId: "channel-0"},
			},
			portID:        "transfer",
			channelID:     "channel-0",
			expectedErr:   false,
			expectedCount: 1,
			expectedResult: []ibcutil.AuthorizedChannel{
				{PortId: "transfer", ChannelId: "channel-0"},
			},
		},
		{
			name: "authorize second channel",
			initial: []ibcutil.AuthorizedChannel{
				{PortId: "transfer", ChannelId: "channel-0"},
			},
			portID:        "oracle",
			channelID:     "channel-1",
			expectedErr:   false,
			expectedCount: 2,
			expectedResult: []ibcutil.AuthorizedChannel{
				{PortId: "transfer", ChannelId: "channel-0"},
				{PortId: "oracle", ChannelId: "channel-1"},
			},
		},
		{
			name:          "empty port_id",
			initial:       []ibcutil.AuthorizedChannel{},
			portID:        "",
			channelID:     "channel-0",
			expectedErr:   true,
			expectedCount: 0,
		},
		{
			name:          "empty channel_id",
			initial:       []ibcutil.AuthorizedChannel{},
			portID:        "transfer",
			channelID:     "",
			expectedErr:   true,
			expectedCount: 0,
		},
		{
			name:          "whitespace-only port_id",
			initial:       []ibcutil.AuthorizedChannel{},
			portID:        "   ",
			channelID:     "channel-0",
			expectedErr:   true,
			expectedCount: 0,
		},
		{
			name:          "whitespace trimming",
			initial:       []ibcutil.AuthorizedChannel{},
			portID:        "  transfer  ",
			channelID:     "  channel-0  ",
			expectedErr:   false,
			expectedCount: 1,
			expectedResult: []ibcutil.AuthorizedChannel{
				{PortId: "transfer", ChannelId: "channel-0"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockChannelStore{channels: tt.initial}
			ctx := context.Background()

			err := ibcutil.AuthorizeChannel(ctx, store, tt.portID, tt.channelID)

			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Len(t, store.channels, tt.expectedCount)
				if tt.expectedResult != nil {
					require.Equal(t, tt.expectedResult, store.channels)
				}
			}
		})
	}
}

func TestIsAuthorizedChannel(t *testing.T) {
	tests := []struct {
		name       string
		channels   []ibcutil.AuthorizedChannel
		portID     string
		channelID  string
		authorized bool
	}{
		{
			name: "authorized channel",
			channels: []ibcutil.AuthorizedChannel{
				{PortId: "transfer", ChannelId: "channel-0"},
			},
			portID:     "transfer",
			channelID:  "channel-0",
			authorized: true,
		},
		{
			name: "unauthorized port",
			channels: []ibcutil.AuthorizedChannel{
				{PortId: "transfer", ChannelId: "channel-0"},
			},
			portID:     "oracle",
			channelID:  "channel-0",
			authorized: false,
		},
		{
			name: "unauthorized channel",
			channels: []ibcutil.AuthorizedChannel{
				{PortId: "transfer", ChannelId: "channel-0"},
			},
			portID:     "transfer",
			channelID:  "channel-1",
			authorized: false,
		},
		{
			name: "multiple authorized channels",
			channels: []ibcutil.AuthorizedChannel{
				{PortId: "transfer", ChannelId: "channel-0"},
				{PortId: "oracle", ChannelId: "channel-1"},
				{PortId: "compute", ChannelId: "channel-2"},
			},
			portID:     "oracle",
			channelID:  "channel-1",
			authorized: true,
		},
		{
			name:       "empty authorized list",
			channels:   []ibcutil.AuthorizedChannel{},
			portID:     "transfer",
			channelID:  "channel-0",
			authorized: false,
		},
		{
			name:       "nil authorized list",
			channels:   nil,
			portID:     "transfer",
			channelID:  "channel-0",
			authorized: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockChannelStore{channels: tt.channels}
			ctx := context.Background()

			result := ibcutil.IsAuthorizedChannel(ctx, store, tt.portID, tt.channelID)
			require.Equal(t, tt.authorized, result)
		})
	}
}

func TestSetAuthorizedChannelsWithValidation(t *testing.T) {
	tests := []struct {
		name           string
		input          []ibcutil.AuthorizedChannel
		expectedErr    bool
		expectedResult []ibcutil.AuthorizedChannel
	}{
		{
			name: "valid channels",
			input: []ibcutil.AuthorizedChannel{
				{PortId: "transfer", ChannelId: "channel-0"},
				{PortId: "oracle", ChannelId: "channel-1"},
			},
			expectedErr: false,
			expectedResult: []ibcutil.AuthorizedChannel{
				{PortId: "transfer", ChannelId: "channel-0"},
				{PortId: "oracle", ChannelId: "channel-1"},
			},
		},
		{
			name: "duplicate channels removed",
			input: []ibcutil.AuthorizedChannel{
				{PortId: "transfer", ChannelId: "channel-0"},
				{PortId: "transfer", ChannelId: "channel-0"},
				{PortId: "oracle", ChannelId: "channel-1"},
			},
			expectedErr: false,
			expectedResult: []ibcutil.AuthorizedChannel{
				{PortId: "transfer", ChannelId: "channel-0"},
				{PortId: "oracle", ChannelId: "channel-1"},
			},
		},
		{
			name: "whitespace trimmed",
			input: []ibcutil.AuthorizedChannel{
				{PortId: "  transfer  ", ChannelId: "  channel-0  "},
			},
			expectedErr: false,
			expectedResult: []ibcutil.AuthorizedChannel{
				{PortId: "transfer", ChannelId: "channel-0"},
			},
		},
		{
			name: "empty port_id rejected",
			input: []ibcutil.AuthorizedChannel{
				{PortId: "", ChannelId: "channel-0"},
			},
			expectedErr: true,
		},
		{
			name: "empty channel_id rejected",
			input: []ibcutil.AuthorizedChannel{
				{PortId: "transfer", ChannelId: ""},
			},
			expectedErr: true,
		},
		{
			name: "whitespace-only port_id rejected",
			input: []ibcutil.AuthorizedChannel{
				{PortId: "   ", ChannelId: "channel-0"},
			},
			expectedErr: true,
		},
		{
			name: "empty list accepted",
			input: []ibcutil.AuthorizedChannel{},
			expectedErr: false,
			expectedResult: []ibcutil.AuthorizedChannel{},
		},
		{
			name:        "nil list accepted",
			input:       nil,
			expectedErr: false,
			expectedResult: []ibcutil.AuthorizedChannel{},
		},
		{
			name: "duplicate after normalization",
			input: []ibcutil.AuthorizedChannel{
				{PortId: "transfer", ChannelId: "channel-0"},
				{PortId: "  transfer  ", ChannelId: "  channel-0  "},
			},
			expectedErr: false,
			expectedResult: []ibcutil.AuthorizedChannel{
				{PortId: "transfer", ChannelId: "channel-0"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockChannelStore{}
			ctx := context.Background()

			err := ibcutil.SetAuthorizedChannelsWithValidation(ctx, store, tt.input)

			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.expectedResult != nil {
					require.Equal(t, tt.expectedResult, store.channels)
				}
			}
		})
	}
}

func TestIsAuthorizedChannel_StoreError(t *testing.T) {
	// Test that IsAuthorizedChannel returns false when store returns error (fail-safe)
	store := &mockChannelStore{
		getErr: context.DeadlineExceeded,
	}
	ctx := context.Background()

	result := ibcutil.IsAuthorizedChannel(ctx, store, "transfer", "channel-0")
	require.False(t, result, "should return false on store error (fail-safe)")
}
