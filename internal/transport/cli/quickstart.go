package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/boi-family/boi-cli/internal/app"
	coreblock "github.com/boi-family/boi-cli/internal/core"
	llmfactory "github.com/boi-family/boi-cli/internal/service/provider/factory"
	"github.com/boi-family/boi-cli/internal/transport/tui/setup"
	"github.com/spf13/cobra"
)

// Seams for tests: the local servers to look for, the hosted-provider wizard,
// qualification and the .env writer.
var (
	quickstartCandidates = localCandidates
	quickstartHosted     = func() error { return RunSetupWizard(false) }
	quickstartQualify    = qualifyProvider
	quickstartSave       = setup.SaveProviders
)

var quickstartCmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Get ready in one go: find a model, check it, start",
	Long: `Guides a first run from nothing to a working Agent:

  1. finds a model: a local server (Ollama, LM Studio, llama.cpp) is detected
     for you, or you can use a hosted provider with an API key
  2. saves the settings in this project's .env (kept out of Git)
  3. checks what the model can do, so the Agent only uses what it proved

Run it again any time to change the model or repeat the check.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		runtime, ok := app.RuntimeFromContext(cmd.Context())
		if !ok || runtime == nil {
			return fmt.Errorf("application runtime is not configured")
		}
		return RunQuickstart(cmd.Context(), runtime, bufio.NewReader(cmd.InOrStdin()), cmd.OutOrStdout())
	},
}

func init() { rootCmd.AddCommand(quickstartCmd) }

// RunQuickstart takes a project from nothing to an Agent whose Provider has
// been checked, asking only what it cannot find out itself.
func RunQuickstart(ctx context.Context, runtime *app.Runtime, in *bufio.Reader, out io.Writer) error {
	fmt.Fprintln(out, "BOI quick start")
	fmt.Fprintln(out)
	if _, err := os.Stat(filepath.Join(runtime.BoiDir, "config.yaml")); os.IsNotExist(err) {
		if err := RunInitAt(runtime.WorkspaceRoot, true); err != nil {
			return fmt.Errorf("prepare the project: %w", err)
		}
	}
	if err := runtime.EnsureWorkspaceState(); err != nil {
		return fmt.Errorf("prepare the project: %w", err)
	}
	loadEnvFiles(runtime.EnvPath)

	if llmfactory.CountProvidersFromEnv() == 0 {
		fmt.Fprintln(out, "Step 1 of 2: find a model")
		if err := chooseProvider(ctx, runtime, in, out); err != nil {
			return err
		}
		loadEnvFiles(runtime.EnvPath)
	}
	configured, err := llmfactory.LoadConfiguredProvidersFromEnv()
	if err != nil || len(configured) == 0 {
		fmt.Fprintln(out, "No model is configured yet. Run 'boi quickstart' again when a model server is running or you have an API key.")
		return nil
	}
	item := configured[0]
	fmt.Fprintf(out, "Model: %s (%s)\n\n", item.Model, item.Name)

	fmt.Fprintln(out, "Step 2 of 2: check what the model can do")
	if len(app.QualifiedProviders(runtime.BoiDir, configured)) == 0 {
		if !confirmQualification(item, in, out) {
			fmt.Fprintf(out, "Skipped. Run 'boi provider qualify %s' before asking the Agent.\n", item.Name)
			return nil
		}
		if err := quickstartQualify(ctx, runtime, item, out, coreblock.DefaultProbeTimeout(item.EndpointClass)); err != nil {
			return fmt.Errorf("check the model: %w", err)
		}
	} else {
		fmt.Fprintln(out, "  Already checked.")
	}
	fmt.Fprintln(out)
	printVerdict(runtime, configured, out)
	return nil
}

// confirmQualification asks before spending requests; a local model costs
// nothing, so it only warns about the time.
func confirmQualification(item llmfactory.ConfiguredProvider, in *bufio.Reader, out io.Writer) bool {
	if item.EndpointClass == "local" {
		fmt.Fprintln(out, "  Sending a few small requests to your local model. The first one loads it, which can take a few minutes.")
		return true
	}
	fmt.Fprint(out, "  This sends a few small paid requests to the provider. Continue? [Y/n] ")
	answer, _ := in.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "" || answer == "y" || answer == "yes"
}

// printVerdict says in plain words what the Agent will be able to do.
func printVerdict(runtime *app.Runtime, configured []llmfactory.ConfiguredProvider, out io.Writer) {
	qualified := app.QualifiedProviders(runtime.BoiDir, configured)
	environment := app.ProviderEnvironment(runtime.BoiDir, qualified)
	switch {
	case len(qualified) == 0:
		fmt.Fprintln(out, "The model did not pass the basic checks, so the Agent cannot use it. See the list above; a larger or instruct-tuned model usually passes.")
	case environment.ToolCalling:
		fmt.Fprintln(out, "Ready. The Agent can read and edit files and run commands in this folder; every change asks you first.")
		fmt.Fprintln(out, "  boi                      chat and work on this project")
		fmt.Fprintln(out, "  boi ask \"your question\"   one read-only answer")
		fmt.Fprintln(out, "  boi env                  what the Agent can use right now")
		if !environment.NativeTools {
			fmt.Fprintln(out, "Note: 'native_tool_schema failed' above is normal. The Agent then talks to the model in plain text, which works the same.")
		}
	default:
		fmt.Fprintln(out, "The model can chat, but it did not prove it can use tools reliably, so the Agent will answer without reading or changing files.")
		fmt.Fprintln(out, "A larger or coder-tuned model usually passes. Run 'boi quickstart' again after changing the model.")
	}
}

type modelChoice struct {
	label    string
	provider setup.ConfiguredProvider
	hosted   bool
	custom   bool
}

// chooseProvider finds a local server for the user, or hands over to the
// hosted-provider wizard, and saves the result.
func chooseProvider(ctx context.Context, runtime *app.Runtime, in *bufio.Reader, out io.Writer) error {
	fmt.Fprintln(out, "  Looking for a model server on this computer...")
	servers := detectLocalServers(ctx, http.DefaultClient, quickstartCandidates)
	var choices []modelChoice
	for _, server := range servers {
		for index, model := range server.Models {
			if index >= 8 {
				break
			}
			choices = append(choices, modelChoice{
				label:    fmt.Sprintf("%s: %s", server.Label, model),
				provider: setup.ConfiguredProvider{Name: "ollama", Endpoint: server.BaseURL, Model: model},
			})
		}
	}
	if len(choices) == 0 {
		fmt.Fprintln(out, "  None found (looked for Ollama, LM Studio and llama.cpp on their usual ports).")
	}
	choices = append(choices,
		modelChoice{label: "A hosted provider with an API key (OpenAI, Anthropic, Cloudflare and others)", hosted: true},
		modelChoice{label: "A local server at another address", custom: true},
	)
	fmt.Fprintln(out)
	for index, item := range choices {
		fmt.Fprintf(out, "  %d) %s\n", index+1, item.label)
	}
	selected, ok := askChoice(in, out, len(choices))
	if !ok {
		return fmt.Errorf("no choice made")
	}
	item := choices[selected]
	switch {
	case item.hosted:
		return quickstartHosted()
	case item.custom:
		provider, err := askLocalServer(ctx, in, out)
		if err != nil {
			return err
		}
		item.provider = provider
	}
	envPath, _, err := quickstartSave([]setup.ConfiguredProvider{item.provider})
	if err != nil {
		return fmt.Errorf("save the settings: %w", err)
	}
	fmt.Fprintf(out, "\n  Saved to %s (kept out of Git).\n\n", envPath)
	return nil
}

// askChoice reads a number from 1 to count; Enter means the first. Three
// invalid answers, or the end of input, give up.
func askChoice(in *bufio.Reader, out io.Writer, count int) (int, bool) {
	for attempt := 0; attempt < 3; attempt++ {
		fmt.Fprintf(out, "\n  Choose [1-%d, Enter = 1]: ", count)
		line, err := in.ReadString('\n')
		answer := strings.TrimSpace(line)
		if answer == "" && err == nil {
			return 0, true
		}
		var number int
		if _, scanErr := fmt.Sscanf(answer, "%d", &number); scanErr == nil && number >= 1 && number <= count {
			return number - 1, true
		}
		if err != nil {
			return 0, false
		}
		fmt.Fprintln(out, "  Please type one of the numbers above.")
	}
	return 0, false
}

// askLocalServer asks for the address of a server that is not on a usual
// port, and lists its models when the name is left empty.
func askLocalServer(ctx context.Context, in *bufio.Reader, out io.Writer) (setup.ConfiguredProvider, error) {
	fmt.Fprint(out, "  Address, for example http://localhost:8080/v1: ")
	address, _ := in.ReadString('\n')
	address = strings.TrimSpace(address)
	parsed, err := url.Parse(address)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		return setup.ConfiguredProvider{}, fmt.Errorf("%q is not an address such as http://localhost:8080/v1", address)
	}
	fmt.Fprint(out, "  Model name (Enter = the first one the server lists): ")
	model, _ := in.ReadString('\n')
	model = strings.TrimSpace(model)
	if model == "" {
		models := listModels(ctx, http.DefaultClient, address)
		if len(models) == 0 {
			return setup.ConfiguredProvider{}, fmt.Errorf("the server at %s did not list any model; type the model name", address)
		}
		model = models[0]
	}
	return setup.ConfiguredProvider{Name: "ollama", Endpoint: address, Model: model}, nil
}
