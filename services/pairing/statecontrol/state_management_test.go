package statecontrol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	cs  = "cs2:4FHRnp:Q4:uqnn"
	cs1 = cs + "1234"
	cs2 = cs + "qwer"
)

func TestProcessStateManager_StartPairing(t *testing.T) {
	psm := new(ProcessStateManager)

	// A new psm should start with no error
	err := psm.StartPairing(cs)
	require.NoError(t, err)

	// A started psm should return an ErrProcessStateManagerAlreadyPairing if another start is attempted
	err = psm.StartPairing(cs)
	require.EqualError(t, err, ErrProcessStateManagerAlreadyPairing.Error())

	// A psm should start without error if the pairing process has been stopped with an error
	psm.StopPairing(cs, err)
	err = psm.StartPairing(cs)
	require.NoError(t, err)

	// A psm should return an error if starting with a conn string that previously succeeded (a nil error)
	psm.StopPairing(cs, nil)
	err = psm.StartPairing(cs)
	require.EqualError(t, err, ErrProcessStateManagerAlreadyPaired(cs).Error())

	// A psm should be able to start with a new connection string if the psm has been stopped
	err = psm.StartPairing(cs1)
	require.NoError(t, err)

	// A started psm should return an ErrProcessStateManagerAlreadyPairing if another start is attempted
	err = psm.StartPairing(cs1)
	require.EqualError(t, err, ErrProcessStateManagerAlreadyPairing.Error())

	// A started psm should return an ErrProcessStateManagerAlreadyPairing if another start is attempted regardless of
	// the given connection string.
	err = psm.StartPairing(cs2)
	require.EqualError(t, err, ErrProcessStateManagerAlreadyPairing.Error())

	// A psm should start without error if the pairing process has been stopped with an error
	psm.StopPairing(cs1, err)
	err = psm.StartPairing(cs2)
	require.NoError(t, err)

	// A psm should return an error if starting with a conn string that previously succeeded (a nil error)
	psm.StopPairing(cs2, nil)
	err = psm.StartPairing(cs2)
	require.EqualError(t, err, ErrProcessStateManagerAlreadyPaired(cs2).Error())
}
