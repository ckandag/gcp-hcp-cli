package ops

import (
	"testing"
)

func TestNewDiagnoseCmd(t *testing.T) {
	cmd := newDiagnoseCmd()

	if cmd.Use != "diagnose <query>" {
		t.Errorf("expected Use='diagnose <query>', got %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected non-empty Short description")
	}

	serviceNameFlag := cmd.Flags().Lookup("service-name")
	if serviceNameFlag == nil {
		t.Fatal("expected --service-name flag")
	}
	if serviceNameFlag.DefValue != "diagnose-agent" {
		t.Errorf("expected default service-name 'diagnose-agent', got %q", serviceNameFlag.DefValue)
	}

	timeoutFlag := cmd.Flags().Lookup("timeout")
	if timeoutFlag == nil {
		t.Fatal("expected --timeout flag")
	}
	if timeoutFlag.DefValue != "3m0s" {
		t.Errorf("expected default timeout '3m0s', got %q", timeoutFlag.DefValue)
	}
}

func TestNewOpsCmd_IncludesDiagnose(t *testing.T) {
	cmd := NewOpsCmd()

	subcommands := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subcommands[sub.Name()] = true
	}

	expected := []string{"get", "logs", "describe", "diagnose", "wf"}
	for _, name := range expected {
		if !subcommands[name] {
			t.Errorf("expected subcommand %q not found", name)
		}
	}
}
