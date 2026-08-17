package cli

import "testing"

func TestRootCommandContract(t *testing.T) {
	t.Parallel()

	expectedCommands := []string{
		"ask",
		"config",
		"doctor",
		"init",
		"memory",
		"model",
		"persona",
		"provider",
		"run",
		"setup",
		"skill",
		"uninstall",
		"upgrade",
		"version",
		"weight",
	}

	for _, name := range expectedCommands {
		name := name
		t.Run(name, func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{name})
			if err != nil {
				t.Fatalf("find command %q: %v", name, err)
			}
			if cmd == rootCmd || cmd.Name() != name {
				t.Fatalf("command %q is not registered", name)
			}
		})
	}
}

func TestAskCommandDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
	}{
		{name: "persona", want: "kamkaew"},
		{name: "steps", want: "15"},
		{name: "verbose", want: "false"},
	}

	for _, tt := range tests {
		flag := askCmd.Flags().Lookup(tt.name)
		if flag == nil {
			t.Fatalf("ask flag %q is not registered", tt.name)
		}
		if flag.DefValue != tt.want {
			t.Fatalf("ask flag %q default = %q, want %q", tt.name, flag.DefValue, tt.want)
		}
	}
}
