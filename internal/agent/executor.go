package agent

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) ExecuteTool(name, args string) (string, error) {
	switch name {
	case "glob":
		return e.runCommand("dir /s /b " + args)
	case "read":
		data, err := os.ReadFile(args)
		if err != nil {
			return "", err
		}
		content := string(data)
		if len(content) > 2000 {
			content = content[:2000] + "..."
		}
		return content, nil
	case "search":
		return e.runCommand("findstr /s /i /c:\"" + args + "\" *.go")
	case "shell":
		return e.runCommand(args)
	case "think":
		return "Analysis complete — proceeding to next step", nil
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func (e *Executor) runCommand(args string) (string, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-Command", args)
	} else {
		cmd = exec.Command("sh", "-c", args)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, string(output))
	}
	return string(output), nil
}
