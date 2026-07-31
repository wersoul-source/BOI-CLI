package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/boi-family/boi-cli/internal/config"
	"github.com/boi-family/boi-cli/internal/persona"
	"github.com/boi-family/boi-cli/internal/skill"
	"github.com/boi-family/boi-cli/internal/workspace"
	"github.com/spf13/cobra"
)

type checkResult struct {
	label  string
	ok     bool
	detail string
	warn   bool
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run health checks on BOI CLI",
	Long:  "Checks Go version, workspace, config, personas, skills, memory, providers, binary, and OS.",
	RunE: func(cmd *cobra.Command, args []string) error {
		var checks []checkResult
		passed := 0
		total := 0

		goVer := runtime.Version()
		goVersionOK := checkGoVersion(goVer)
		total++
		detail := goVer
		if goVersionOK {
			passed++
		}
		checks = append(checks, checkResult{label: "Go", ok: goVersionOK, detail: detail})

		root, err := workspace.DetectRoot()
		wsOK := err == nil
		if !wsOK {
			root, _ = os.Getwd()
		}
		if wsOK {
			boiDir := workspace.GetBoiDir(root)
			if _, e := os.Stat(boiDir); e != nil {
				wsOK = false
			}
		}
		total++
		if wsOK {
			passed++
		}
		checks = append(checks, checkResult{label: "Workspace", ok: wsOK, detail: root})

		configPath := filepath.Join(workspace.GetBoiDir(root), "config.yaml")
		_, err = config.LoadFrom(configPath)
		cfgOK := err == nil
		personaCount := 0
		if reg, personasErr := persona.Load(filepath.Join(workspace.GetBoiDir(root), "personas")); personasErr == nil && reg != nil {
			personaCount = reg.Count()
		}
		total++
		if cfgOK {
			passed++
		}
		checks = append(checks, checkResult{label: "Config", ok: cfgOK, detail: fmt.Sprintf("%d personas", personaCount)})

		skillDir := filepath.Join(workspace.GetBoiDir(root), "skills")
		skills, skillErr := skill.LoadDir(skillDir)
		skillCount := 0
		if skillErr == nil {
			skillCount = len(skills)
		}
		skillOK := skillErr == nil
		total++
		if skillOK {
			passed++
		}
		checks = append(checks, checkResult{label: "Skills", ok: skillOK, detail: fmt.Sprintf("%d loaded", skillCount)})

		dbDir := filepath.Join(workspace.GetBoiDir(root), "memory")
		memOK := true
		memCount := 0
		if entries, e := os.ReadDir(dbDir); e == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
					memCount++
				}
			}
		} else {
			memOK = false
		}
		total++
		if memOK {
			passed++
		}
		checks = append(checks, checkResult{label: "Memory", ok: memOK, detail: fmt.Sprintf("%d entries", memCount)})

		providerCount := countPSCProviders()
		provOK := providerCount > 0
		provWarn := !provOK
		total++
		provDetail := fmt.Sprintf("%d configured", providerCount)
		if provOK {
			passed++
		}
		checks = append(checks, checkResult{
			label:  "Providers",
			ok:     provOK,
			detail: provDetail,
			warn:   provWarn,
		})

		total++
		passed++
		checks = append(checks, checkResult{label: "Binary", ok: true, detail: fmt.Sprintf("boi v%s", Version)})

		osArch := runtime.GOOS + "/" + runtime.GOARCH
		total++
		passed++
		checks = append(checks, checkResult{label: "OS", ok: true, detail: osArch})

		fmt.Println("BOI CLI Doctor")
		fmt.Println("\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500")
		for _, c := range checks {
			icon := "\u2705"
			if c.warn {
				icon = "\u26a0\ufe0f"
			} else if !c.ok {
				icon = "\u274c"
			}
			if c.detail != "" {
				fmt.Printf("%s %s: %s\n", icon, c.label, c.detail)
			} else {
				fmt.Printf("%s %s\n", icon, c.label)
			}
		}

		fmt.Println()
		if passed == total {
			fmt.Printf("All checks passed! (%d/%d)\n", passed, total)
		} else {
			fixMsg := ""
			if !provOK {
				fixMsg = "Set PSC_* in .env"
			}
			if !wsOK || !cfgOK {
				if fixMsg != "" {
					fixMsg += " & run 'boi init'"
				} else {
					fixMsg = "Run 'boi init'"
				}
			}
			fmt.Printf("All checks passed! (%d/%d)", passed, total)
			if fixMsg != "" {
				fmt.Printf(" \u2014 %s to fix issues.", fixMsg)
			}
			fmt.Println()
		}

		if !provOK {
			fmt.Printf("\nTip: Set PSC_* in .env to enable AI providers.\n")
		}

		return nil
	},
}

func checkGoVersion(v string) bool {
	v = strings.TrimPrefix(v, "go")
	parts := strings.Split(v, ".")
	if len(parts) < 2 {
		return false
	}
	var major, minor int
	fmt.Sscanf(parts[0], "%d", &major)
	fmt.Sscanf(parts[1], "%d", &minor)
	return major > 1 || (major == 1 && minor >= 24)
}

func countPSCProviders() int {
	count := 0
	for i := 1; i <= 4; i++ {
		name := os.Getenv(fmt.Sprintf("PSC_%d_NAME", i))
		key := os.Getenv(fmt.Sprintf("PSC_%d_API_KEY", i))
		if name != "" && key != "" {
			count++
		}
	}
	return count
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
