package seed

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neu-software-practice/software-practice-backend/internal/model"
	"github.com/neu-software-practice/software-practice-backend/internal/pkg/hash"
	"github.com/neu-software-practice/software-practice-backend/internal/testutil"
)

func TestRun_SeedsBaseDataAndIsIdempotent(t *testing.T) {
	db := testutil.NewDB(t)

	require.NoError(t, Run(db, "Passw0rd!"))

	count := func(m interface{}) int64 {
		var n int64
		require.NoError(t, db.Model(m).Count(&n).Error)
		return n
	}
	assert.Equal(t, int64(7), count(&model.Department{}), "7 departments incl. root")
	assert.Equal(t, int64(7), count(&model.Employee{}), "one account per role + root")
	assert.Equal(t, int64(3), count(&model.RegistLevel{}))
	assert.Equal(t, int64(3), count(&model.SettleCategory{}))
	assert.Equal(t, int64(8), count(&model.MedicalTechnology{}))
	assert.Equal(t, int64(5), count(&model.Disease{}))
	assert.Equal(t, int64(5), count(&model.DrugInfo{}))

	// Passwords are stored as verifiable bcrypt hashes, never plaintext.
	var doctor model.Employee
	require.NoError(t, db.Where("username = ?", "doctor").First(&doctor).Error)
	assert.NotEqual(t, "Passw0rd!", doctor.Password)
	assert.True(t, hash.Verify(doctor.Password, "Passw0rd!"))
	assert.NotNil(t, doctor.RegistLevelID, "doctor must carry a registration level")

	// Re-running must not duplicate rows.
	require.NoError(t, Run(db, "Passw0rd!"))
	assert.Equal(t, int64(7), count(&model.Employee{}))
	assert.Equal(t, int64(8), count(&model.MedicalTechnology{}))
}
