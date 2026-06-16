package hash_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neu-software-practice/software-practice-backend/internal/pkg/hash"
)

func TestPasswordRoundTrip(t *testing.T) {
	hashed, err := hash.Password("Passw0rd!")
	require.NoError(t, err)
	assert.NotEqual(t, "Passw0rd!", hashed)
	assert.True(t, hash.Verify(hashed, "Passw0rd!"))
	assert.False(t, hash.Verify(hashed, "wrong"))
}

func TestPassword_SaltedHashesDiffer(t *testing.T) {
	a, err := hash.Password("same")
	require.NoError(t, err)
	b, err := hash.Password("same")
	require.NoError(t, err)
	assert.NotEqual(t, a, b, "bcrypt must use a random salt per hash")
}

func TestVerify_InvalidHash(t *testing.T) {
	assert.False(t, hash.Verify("not-a-bcrypt-hash", "x"))
}
