package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOptimizeCmd(t *testing.T) {
	cmd := NewOptimizeCmd()

	assert.Equal(t, "optimize", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)
}

func TestNewOptimizeCmd_Flags(t *testing.T) {
	cmd := NewOptimizeCmd()

	// Check all flags exist
	flags := []string{
		"layer",
		"target",
		"dry-run",
		"diff",
		"output",
		"force",
		"deterministic",
		"verbose",
		"no-cache",
	}

	for _, flag := range flags {
		f := cmd.Flags().Lookup(flag)
		require.NotNil(t, f, "flag %q should exist", flag)
	}
}

func TestNewOptimizeCmd_FlagDefaults(t *testing.T) {
	cmd := NewOptimizeCmd()

	// Check default values
	layer, _ := cmd.Flags().GetString("layer")
	assert.Equal(t, "merged", layer)

	target, _ := cmd.Flags().GetInt("target")
	assert.Equal(t, 0, target)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	assert.False(t, dryRun)

	deterministic, _ := cmd.Flags().GetBool("deterministic")
	assert.False(t, deterministic)
}

func TestNewOptimizeCmd_ShortFlags(t *testing.T) {
	cmd := NewOptimizeCmd()

	// Check short flags exist
	shortFlags := map[string]string{
		"o": "output",
		"v": "verbose",
	}

	for short, long := range shortFlags {
		f := cmd.Flags().ShorthandLookup(short)
		require.NotNil(t, f, "short flag %q should exist", short)
		assert.Equal(t, long, f.Name)
	}
}
