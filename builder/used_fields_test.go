package builder

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewUsedFields(t *testing.T) {
	u := newUsedFields()
	require.NotNil(t, u)
	require.NotNil(t, u.full)
	require.NotNil(t, u.top)
	require.Len(t, u.full, 0)
	require.Len(t, u.top, 0)
}

func TestUsedFields_Add(t *testing.T) {
	u := newUsedFields()

	u.add("user.name")
	u.add("profile")

	_, ok := u.full["user.name"]
	require.True(t, ok)

	_, ok = u.full["profile"]
	require.True(t, ok)

	_, ok = u.top["user"]
	require.True(t, ok)

	_, ok = u.top["profile"]
	require.True(t, ok)

	require.Len(t, u.full, 2)
	require.Len(t, u.top, 2)
}

func TestUsedFields_HasFullKey(t *testing.T) {
	u := newUsedFields()
	u.add("user.name")

	require.True(t, u.hasFullKey("user.name"))
	require.False(t, u.hasFullKey("user.email"))
}

func TestUsedFields_HasTopLevelKey(t *testing.T) {
	u := newUsedFields()
	u.add("user.name")
	u.add("settings.theme")

	require.True(t, u.hasTopLevelKey("user"))
	require.True(t, u.hasTopLevelKey("settings"))
	require.False(t, u.hasTopLevelKey("profile"))
}

func TestUsedFields_AddDuplicate(t *testing.T) {
	u := newUsedFields()
	u.add("user.name")
	u.add("user.name")

	require.True(t, u.hasFullKey("user.name"))
	require.True(t, u.hasTopLevelKey("user"))
	require.Len(t, u.full, 1)
	require.Len(t, u.top, 1)
}

func TestUsedFields_Len(t *testing.T) {
	u := newUsedFields()

	u.add("user.name")
	u.add("profile")
	u.add("user.name")
	u.add("profile")

	require.Equal(t, u.len(), 2)
}
