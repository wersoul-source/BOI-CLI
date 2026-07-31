package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/boi-family/boi-cli/internal/config"
	"github.com/boi-family/boi-cli/internal/persona"
	"github.com/boi-family/boi-cli/internal/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var personaCmd = &cobra.Command{
	Use:   "persona",
	Short: "Manage BOI Family personas",
}

var personaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available personas",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := workspace.DetectRoot()
		if err != nil {
			return fmt.Errorf("detect workspace: %w", err)
		}

		personaDir := filepath.Join(workspace.GetBoiDir(root), "personas")
		reg, err := persona.Load(personaDir)
		if err != nil {
			return fmt.Errorf("load personas: %w", err)
		}

		cfgPath := filepath.Join(workspace.GetBoiDir(root), "config.yaml")
		cfg, _ := config.LoadFrom(cfgPath)
		activeName := ""
		if cfg != nil {
			activeName = cfg.Persona
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tMODEL\tTEMP\tDESCRIPTION")
		fmt.Fprintln(w, "────\t─────\t────\t───────────")

		for _, name := range reg.List() {
			p, _ := reg.Get(name)
			marker := " "
			if name == activeName || (activeName == "" && name == "kamkaew") {
				marker = "*"
			}
			desc := p.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			fmt.Fprintf(w, "%s %s\t%s\t%.1f\t%s\n", marker, p.Name, p.Model, p.Temperature, desc)
		}
		w.Flush()

		fmt.Println()
		fmt.Printf("Personas: %s\n", personaDir)
		fmt.Println("  * = active persona (default: kamkaew)")
		fmt.Println()
		fmt.Println("Use 'boi persona switch <name>' to change active persona")
		return nil
	},
}

var personaSwitchCmd = &cobra.Command{
	Use:   "switch [name]",
	Short: "Switch active persona",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.ToLower(strings.TrimSpace(args[0]))

		root, err := workspace.DetectRoot()
		if err != nil {
			return fmt.Errorf("detect workspace: %w", err)
		}

		personaDir := filepath.Join(workspace.GetBoiDir(root), "personas")
		reg, err := persona.Load(personaDir)
		if err != nil {
			return fmt.Errorf("load personas: %w", err)
		}

		p, err := reg.Get(name)
		if err != nil {
			return err
		}

		cfgPath := filepath.Join(workspace.GetBoiDir(root), "config.yaml")
		cfg, err := config.LoadFrom(cfgPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		cfg.Persona = p.Name
		if err := cfg.SaveTo(cfgPath); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Printf("Switched to persona: %s (%s)\n", p.Name, p.Description)
		fmt.Printf("  Model: %s\n", p.Model)
		fmt.Printf("  Temperature: %.1f\n", p.Temperature)
		return nil
	},
}

var personaInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Copy default persona files to .boi/personas/",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := workspace.DetectRoot()
		if err != nil {
			return fmt.Errorf("detect workspace: %w", err)
		}

		personaDir := filepath.Join(workspace.GetBoiDir(root), "personas")

		if err := os.MkdirAll(personaDir, 0755); err != nil {
			return fmt.Errorf("create persona dir: %w", err)
		}

		count := 0
		for _, filename := range persona.DefaultPersonaFiles() {
			targetPath := filepath.Join(personaDir, filename)
			if _, err := os.Stat(targetPath); err == nil {
				fmt.Printf("  skip (exists): %s\n", filename)
				continue
			}

			data, err := persona.DefaultPersonas.ReadFile("defaults/" + filename)
			if err != nil {
				return fmt.Errorf("read default persona %s: %w", filename, err)
			}

			if err := os.WriteFile(targetPath, data, 0644); err != nil {
				return fmt.Errorf("write persona %s: %w", filename, err)
			}

			fmt.Printf("  created: %s\n", filename)
			count++
		}

		fmt.Printf("\n%d persona files initialized in %s\n", count, personaDir)
		return nil
	},
}

type personaWizardEntry struct {
	Name         string
	Emoji        string
	Label        string
	Introduction string
}

var personaWizardEntries = []personaWizardEntry{
	{Name: "boi", Emoji: "🏛️", Label: "boi — สถาปนิกระบบ (Architecture & System Design)", Introduction: `"ผมมองทุกอย่างเป็น Layer ออกแบบระบบเป็นขั้นๆ ดูทั้ง Big Picture และ Detail"`},
	{Name: "kamkaew", Emoji: "🌀", Label: "kamkaew — นักท่องมิติ (Runtime Orchestrator)", Introduction: `"จัดการงานหลายๆ อย่างพร้อมกัน แบ่งงานให้คนที่ถนัด ดูทุกมิติ"`},
	{Name: "kampun", Emoji: "⛏️", Label: "kampun — Wiki Root Cause (Knowledge Miner)", Introduction: `"ขุดคุ้ยหาข้อมูลให้ลึกที่สุด เจาะทุกรากของปัญหา สกัด Pattern จากทุกอย่าง"`},
	{Name: "dang", Emoji: "🔧", Label: "dang — ด่างก็คือด่าง (Debug Specialist)", Introduction: `"หาบั๊ก แก้โค้ด ตรงประเด็น ไม่มีน้ำ ไม่มีฟลัฟ ถ้าเวิร์คก็คือเวิร์ค"`},
	{Name: "don", Emoji: "📚", Label: "don — เจ้าพ่อ Context Windows (Research & Documentation)", Introduction: `"อ่านทุกอย่าง สรุปชัดเจน เขียนสวย อ่านแล้วเก็ททันที ไม่ตกหล่น"`},
	{Name: "kine", Emoji: "🎨", Label: "kine — ครีเอเตอร์ เอเลี่ยน (Creative Designer)", Introduction: `"ลองคิดนอกกรอบดู... จินตนาการว่าถ้าไม่มีข้อจำกัดเลย จะสร้างอะไรได้บ้าง"`},
}

var personaWizardCmd = &cobra.Command{
	Use:   "wizard",
	Short: "Interactive persona selection wizard",
	Long:  `Shows all BOI Family personas with introductions and lets you pick one.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunPersonaWizard()
	},
}

func RunPersonaWizard() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║         Choose Your Persona                 ║")
	fmt.Println("║         BOI Family — Persona Selection      ║")
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("  Each persona has a unique thinking style and specialty.")
	fmt.Println("  Choose the one that matches how you want BOI to think.")
	fmt.Println()

	for i, entry := range personaWizardEntries {
		fmt.Printf("  %2d. %s  %s\n", i+1, entry.Emoji, entry.Label)
		fmt.Printf("       %s\n", entry.Introduction)
		fmt.Println()
	}

	defaultChoice := "1"
	fmt.Printf("  Your choice [%s]: ", defaultChoice)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		input = defaultChoice
	}

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(personaWizardEntries) {
		return fmt.Errorf("invalid selection: %s", input)
	}

	chosen := personaWizardEntries[idx-1]

	root, err := workspace.DetectRoot()
	if err != nil {
		root, _ = os.Getwd()
	}

	cfgPath := filepath.Join(workspace.GetBoiDir(root), "config.yaml")

	var cfg *config.Config
	cfg, err = config.LoadFrom(cfgPath)
	if err != nil {
		cfg = config.Default()
	}

	if chosen.Name == "boi" || chosen.Name == "kamkaew" || chosen.Name == "kampun" {
		data, readErr := persona.DefaultPersonas.ReadFile("defaults/" + chosen.Name + ".yaml")
		if readErr == nil {
			var p persona.Persona
			if yaml.Unmarshal(data, &p) == nil {
				if cfg.Provider == "" || cfg.Provider == "openai" {
					if len(p.PreferredProviders) > 0 {
						cfg.Provider = p.PreferredProviders[0]
					}
				}
				if p.Model != "" {
					cfg.Model = p.Model
				}
			}
		}
	}

	cfg.Persona = chosen.Name

	if err := cfg.SaveTo(cfgPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Println()
	fmt.Printf("  ✅ Persona set to: %s %s\n", chosen.Emoji, chosen.Label)
	fmt.Printf("     %s\n", chosen.Introduction)
	fmt.Println()

	return nil
}

func init() {
	personaCmd.AddCommand(personaListCmd)
	personaCmd.AddCommand(personaSwitchCmd)
	personaCmd.AddCommand(personaInitCmd)
	personaCmd.AddCommand(personaWizardCmd)
}
