package discovery

import (
	"errors"
	"testing"

	"cosmossdk.io/log"
	"github.com/stretchr/testify/require"
)

type zeroReader struct{}

func (zeroReader) Read(b []byte) (int, error) {
	for i := range b {
		b[i] = 0
	}
	return len(b), nil
}

type failingReader struct{}

func (failingReader) Read(_ []byte) (int, error) {
	return 0, errors.New("entropy unavailable")
}

func TestSecureIntnUsesEntropySource(t *testing.T) {
	ab := &AddressBook{
		logger:     log.NewNopLogger(),
		randReader: zeroReader{},
	}

	val, err := ab.secureIntn(10)
	require.NoError(t, err)
	require.Equal(t, 0, val)
}

func TestSecureIntnFallbackOnEntropyFailure(t *testing.T) {
	ab := &AddressBook{
		logger:     log.NewNopLogger(),
		randReader: failingReader{},
	}

	val, err := ab.secureIntn(10)
	require.Error(t, err)
	require.Equal(t, 0, val)
}

func TestSecureIntnHandlesZeroRange(t *testing.T) {
	ab := &AddressBook{
		logger:     log.NewNopLogger(),
		randReader: zeroReader{},
	}

	val, err := ab.secureIntn(0)
	require.NoError(t, err)
	require.Equal(t, 0, val)
}
