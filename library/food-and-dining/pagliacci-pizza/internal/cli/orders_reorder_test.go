// Copyright 2026 trevin-chow. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"testing"
)

func TestOrdersReorderCmd_RegistersFlags(t *testing.T) {
	cmd := newOrdersReorderCmd(&rootFlags{})
	if cmd.Flags().Lookup("last") == nil {
		t.Errorf("--last flag missing")
	}
	if cmd.Flags().Lookup("clone-only") == nil {
		t.Errorf("--clone-only flag missing")
	}
	if cmd.Flags().Lookup("send") == nil {
		t.Errorf("--send flag missing")
	}
}

func TestOrdersReorderCmd_RequiresArgsOrLast(t *testing.T) {
	cmd := newOrdersReorderCmd(&rootFlags{})
	cmd.SetArgs([]string{}) // no args, no --last
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	err := cmd.Execute()
	if err == nil {
		t.Errorf("expected error when neither --last nor positional orderId is supplied")
	}
}
