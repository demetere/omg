package omg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Unit tests that don't require Docker/testcontainers

func TestRegister(t *testing.T) {
	Reset()

	m := Migration{
		Version: "20240101000000",
		Name:    "test_migration",
		Up:      nil,
		Down:    nil,
	}

	Register(m)

	all := GetAll()
	assert.Len(t, all, 1)
	assert.Equal(t, "20240101000000", all[0].Version)
	assert.Equal(t, "test_migration", all[0].Name)
}

func TestGetAllSortsVersions(t *testing.T) {
	Reset()

	// Register in reverse order
	Register(Migration{Version: "20240103000000", Name: "third"})
	Register(Migration{Version: "20240101000000", Name: "first"})
	Register(Migration{Version: "20240102000000", Name: "second"})

	all := GetAll()

	assert.Len(t, all, 3)
	assert.Equal(t, "20240101000000", all[0].Version)
	assert.Equal(t, "20240102000000", all[1].Version)
	assert.Equal(t, "20240103000000", all[2].Version)
}

func TestReset(t *testing.T) {
	Reset() // Clear any previous state

	Register(Migration{Version: "20240101000000", Name: "test"})
	assert.Len(t, GetAll(), 1)

	Reset()
	assert.Len(t, GetAll(), 0)
}

func TestGetAllReturnsCopy(t *testing.T) {
	Reset()
	Register(Migration{Version: "20240101000000", Name: "first"})

	all1 := GetAll()
	all2 := GetAll()

	// Should be separate slices
	assert.Equal(t, all1, all2)

	// Modifying one shouldn't affect the other
	all1[0].Name = "modified"
	assert.NotEqual(t, all1[0].Name, all2[0].Name)
}
