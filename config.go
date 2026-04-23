package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/sashabaranov/go-openai"
	"gopkg.in/yaml.v3"
)

// ProgConfig represents program configuration
type ProgConfig struct {
	// LocAI configuration
	LocAIURL             string `yaml:"LocAIURL"`
	LocAIModel           string `yaml:"LocAIModel"`
	LocAIAPIKey          string `yaml:"LocAIAPIKey"`
	LocAICandidateCount  *int32 `yaml:"LocAICandidateCount"`
	LocAIPureResponse    bool   `yaml:"LocAIPureResponse"`
	LocAIMaxOutputTokens *int32 `yaml:"LocAIMaxOutputTokens"`

	// Markdown configuration
	MarkdownPromptResponseFile       string `yaml:"MarkdownPromptResponseFile"`
	MarkdownOutput                   bool   `yaml:"MarkdownOutput"`
	MarkdownOutputApplication        string
	MarkdownOutputApplicationMacOS   string `yaml:"MarkdownOutputApplicationMacOS"`
	MarkdownOutputApplicationLinux   string `yaml:"MarkdownOutputApplicationLinux"`
	MarkdownOutputApplicationWindows string `yaml:"MarkdownOutputApplicationWindows"`
	MarkdownOutputApplicationOther   string `yaml:"MarkdownOutputApplicationOther"`
	MarkdownHistory                  bool   `yaml:"MarkdownHistory"`
	MarkdownHistoryDirectory         string `yaml:"MarkdownHistoryDirectory"`

	// ANSI configuration
	AnsiRendering          bool   `yaml:"AnsiRendering"`
	AnsiPromptResponseFile string `yaml:"AnsiPromptResponseFile"`
	AnsiOutput             bool   `yaml:"AnsiOutput"`
	AnsiOutputLineLength   int    `yaml:"AnsiOutputLineLength"`
	AnsiHistory            bool   `yaml:"AnsiHistory"`
	AnsiHistoryDirectory   string `yaml:"AnsiHistoryDirectory"`
	AnsiOutputTheme        string `yaml:"AnsiOutputTheme"`

	// HTML configuration
	HTMLRendering                bool   `yaml:"HTMLRendering"`
	HTMLPromptResponseFile       string `yaml:"HTMLPromptResponseFile"`
	HTMLOutput                   bool   `yaml:"HTMLOutput"`
	HTMLOutputApplication        string
	HTMLOutputApplicationMacOS   string              `yaml:"HTMLOutputApplicationMacOS"`
	HTMLOutputApplicationLinux   string              `yaml:"HTMLOutputApplicationLinux"`
	HTMLOutputApplicationWindows string              `yaml:"HTMLOutputApplicationWindows"`
	HTMLOutputApplicationOther   string              `yaml:"HTMLOutputApplicationOther"`
	HTMLHistory                  bool                `yaml:"HTMLHistory"`
	HTMLHistoryDirectory         string              `yaml:"HTMLHistoryDirectory"`
	HTMLMaxLengthTitle           int                 `yaml:"HTMLMaxLengthTitle"`
	HTMLReplaceElements          []map[string]string `yaml:"HTMLReplaceElements"`
	HTMLHeader                   string              `yaml:"HTMLHeader"`
	HTMLFooter                   string              `yaml:"HTMLFooter"`

	// Input configuration
	InputFromTerminal  bool   `yaml:"InputFromTerminal"`
	InputFromFile      bool   `yaml:"InputFromFile"`
	InputFile          string `yaml:"InputFile"`
	InputFromLocalhost bool   `yaml:"InputFromLocalhost"`
	InputLocalhostPort int    `yaml:"InputLocalhostPort"`

	// Notification configuration
	NotifyPrompt                     bool `yaml:"NotifyPrompt"`
	NotifyPromptApplication          string
	NotifyPromptApplicationMacOS     string `yaml:"NotifyPromptApplicationMacOS"`
	NotifyPromptApplicationLinux     string `yaml:"NotifyPromptApplicationLinux"`
	NotifyPromptApplicationWindows   string `yaml:"NotifyPromptApplicationWindows"`
	NotifyPromptApplicationOther     string `yaml:"NotifyPromptApplicationOther"`
	NotifyResponse                   bool   `yaml:"NotifyResponse"`
	NotifyResponseApplication        string
	NotifyResponseApplicationMacOS   string `yaml:"NotifyResponseApplicationMacOS"`
	NotifyResponseApplicationLinux   string `yaml:"NotifyResponseApplicationLinux"`
	NotifyResponseApplicationWindows string `yaml:"NotifyResponseApplicationWindows"`
	NotifyResponseApplicationOther   string `yaml:"NotifyResponseApplicationOther"`

	// System instruction
	SystemInstructionFile string `yaml:"SystemInstructionFile"`
}

// progConfig contains program configuration
var progConfig = ProgConfig{}

/*
loadConfiguration loads program configuration from a YAML file. It reads the specified YAML file,
unmarshals it into the global `progConfig` struct, performs extensive validation checks on the loaded
values and sets OS-specific configurations (e.g., application paths). It returns an error if reading,
unmarshalling, validation, or secret retrieval fails.
*/
func loadConfiguration(configFile string) error {
	operatingSystem := runtime.GOOS

	source, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("error [%w] reading configuration file", err)
	}
	err = yaml.Unmarshal(source, &progConfig)
	if err != nil {
		return fmt.Errorf("error [%w] unmarshalling configuration file", err)
	}

	// local AI
	if progConfig.LocAICandidateCount == nil || *progConfig.LocAICandidateCount <= 0 {
		return fmt.Errorf("empty or invalid LocAICandidateCount not allowed")
	}

	// markdown
	if progConfig.MarkdownPromptResponseFile == "" {
		return fmt.Errorf("empty MarkdownPromptResponseFile not allowed")
	}
	switch operatingSystem {
	case "darwin":
		progConfig.MarkdownOutputApplication = progConfig.MarkdownOutputApplicationMacOS
	case "linux":
		progConfig.MarkdownOutputApplication = progConfig.MarkdownOutputApplicationLinux
	case "windows":
		progConfig.MarkdownOutputApplication = progConfig.MarkdownOutputApplicationWindows
	default:
		progConfig.MarkdownOutputApplication = progConfig.MarkdownOutputApplicationOther
	}
	if progConfig.MarkdownOutput && progConfig.MarkdownOutputApplication == "" {
		return fmt.Errorf("empty operating system specific MarkdownOutputApplication not allowed")
	}
	if progConfig.MarkdownHistory && progConfig.MarkdownHistoryDirectory == "" {
		return fmt.Errorf("empty MarkdownHistoryDirectory not allowed")
	}

	// ansi
	if progConfig.AnsiRendering && progConfig.AnsiPromptResponseFile == "" {
		return fmt.Errorf("empty AnsiPromptResponseFile not allowed")
	}
	if progConfig.AnsiHistory && progConfig.AnsiHistoryDirectory == "" {
		return fmt.Errorf("empty AnsiHistoryDirectory not allowed")
	}

	// html
	if progConfig.HTMLRendering && progConfig.HTMLPromptResponseFile == "" {
		return fmt.Errorf("empty HTMLPromptResponseFile not allowed")
	}
	switch operatingSystem {
	case "darwin":
		progConfig.HTMLOutputApplication = progConfig.HTMLOutputApplicationMacOS
	case "linux":
		progConfig.HTMLOutputApplication = progConfig.HTMLOutputApplicationLinux
	case "windows":
		progConfig.HTMLOutputApplication = progConfig.HTMLOutputApplicationWindows
	default:
		progConfig.HTMLOutputApplication = progConfig.HTMLOutputApplicationOther
	}
	if progConfig.HTMLOutput && progConfig.HTMLOutputApplication == "" {
		return fmt.Errorf("empty operating system specific HTMLOutputApplication not allowed")
	}
	if progConfig.HTMLHistory && progConfig.HTMLHistoryDirectory == "" {
		return fmt.Errorf("empty HTMLHistoryDirectory not allowed")
	}

	// input
	if progConfig.InputFromFile && progConfig.InputFile == "" {
		return fmt.Errorf("empty InputFile not allowed")
	}

	// notification
	switch operatingSystem {
	case "darwin":
		progConfig.NotifyPromptApplication = progConfig.NotifyPromptApplicationMacOS
	case "linux":
		progConfig.NotifyPromptApplication = progConfig.NotifyPromptApplicationLinux
	case "windows":
		progConfig.NotifyPromptApplication = progConfig.NotifyPromptApplicationWindows
	default:
		progConfig.NotifyPromptApplication = progConfig.NotifyPromptApplicationOther
	}
	if progConfig.NotifyPrompt && progConfig.NotifyPromptApplication == "" {
		return fmt.Errorf("empty operating system specific NotifyPromptApplication not allowed")
	}
	switch operatingSystem {
	case "darwin":
		progConfig.NotifyResponseApplication = progConfig.NotifyResponseApplicationMacOS
	case "linux":
		progConfig.NotifyResponseApplication = progConfig.NotifyResponseApplicationLinux
	case "windows":
		progConfig.NotifyResponseApplication = progConfig.NotifyResponseApplicationWindows
	default:
		progConfig.NotifyResponseApplication = progConfig.NotifyResponseApplicationOther
	}
	if progConfig.NotifyResponse && progConfig.NotifyResponseApplication == "" {
		return fmt.Errorf("empty operating system specific NotifyResponseApplication not allowed")
	}

	return nil
}

/*
showConfiguration shows / prints the loaded program configuration to the console. It displays the current
program configuration settings to the user in the console for review.
*/
func showConfiguration() {
	fmt.Printf("\nInput from:\n")
	if progConfig.InputFromTerminal {
		fmt.Printf("  Terminal  : yes\n")
	}
	if progConfig.InputFromFile {
		fmt.Printf("  File      : %v\n", progConfig.InputFile)
	}
	if progConfig.InputFromLocalhost {
		fmt.Printf("  localhost : %v (port)\n", progConfig.InputLocalhostPort)
	}

	fmt.Printf("\nRendering:\n")
	fmt.Printf("  Markdown : %v\n", progConfig.MarkdownPromptResponseFile)
	if progConfig.AnsiRendering {
		fmt.Printf("  Ansi     : %v\n", progConfig.AnsiPromptResponseFile)
	}
	if progConfig.HTMLRendering {
		fmt.Printf("  HTML     : %v\n", progConfig.HTMLPromptResponseFile)
	}

	fmt.Printf("\nHistory:\n")
	if progConfig.MarkdownHistory {
		fmt.Printf("  Markdown : %v\n", progConfig.MarkdownHistoryDirectory)
	}
	if progConfig.AnsiHistory {
		fmt.Printf("  Ansi     : %v\n", progConfig.AnsiHistoryDirectory)
	}
	if progConfig.HTMLHistory {
		fmt.Printf("  HTML     : %v\n", progConfig.HTMLHistoryDirectory)
	}

	fmt.Printf("\nOutput:\n")
	if progConfig.AnsiOutput {
		fmt.Printf("  Terminal : yes\n")
	}
	if progConfig.MarkdownOutput {
		fmt.Printf("  Markdown : execute application\n")
	}
	if progConfig.HTMLOutput {
		fmt.Printf("  HTML     : execute application\n")
	}
}

/*
initializeProgram performs program initialization tasks. It sets up the program environment, including
creating necessary directories and writing assets for HTML output.
*/
func initializeProgram() error {
	var err error

	// create history directories
	if progConfig.MarkdownHistory {
		err = os.Mkdir(progConfig.MarkdownHistoryDirectory, 0750)
		if err != nil && !os.IsExist(err) {
			return fmt.Errorf("error [%v] at os.Mkdir()", err)
		}
	}
	if progConfig.AnsiHistory {
		err = os.Mkdir(progConfig.AnsiHistoryDirectory, 0750)
		if err != nil && !os.IsExist(err) {
			return fmt.Errorf("error [%v] at os.Mkdir()", err)
		}
	}
	if progConfig.HTMLHistory {
		err = os.Mkdir(progConfig.HTMLHistoryDirectory, 0750)
		if err != nil && !os.IsExist(err) {
			return fmt.Errorf("error [%v] at os.Mkdir()", err)
		}

		// 'assets' in history directory (to render HTML files in history directory)
		directory := progConfig.HTMLHistoryDirectory + "/assets"
		if !dirExists(directory) {
			err = os.Mkdir(directory, 0750)
			if err != nil && !os.IsExist(err) {
				return fmt.Errorf("error [%v] at os.Mkdir()", err)
			}
			if err := writeAssets(progConfig.HTMLHistoryDirectory); err != nil {
				return err
			}
		}
	}
	return nil
}

/*
generateOpenAIModelConfig generates a configuration object for the local AI model.
*/
func generateOpenAIModelConfig() (openai.ChatCompletionRequest, error) {
	req := openai.ChatCompletionRequest{
		Model: progConfig.LocAIModel,
	}

	// configure AI model parameters
	if progConfig.LocAICandidateCount != nil {
		req.N = int(*progConfig.LocAICandidateCount)
	}

	if progConfig.LocAIMaxOutputTokens != nil && *progConfig.LocAIMaxOutputTokens > 0 {
		req.MaxTokens = int(*progConfig.LocAIMaxOutputTokens)
	}

	// load system instruction from file
	if progConfig.SystemInstructionFile != "" {
		sysPromptBytes, err := os.ReadFile(progConfig.SystemInstructionFile)
		if err != nil {
			return req, fmt.Errorf("error [%v] reading system instruction file [%s]", err, progConfig.SystemInstructionFile)
		}
		finalSystemInstruction = string(sysPromptBytes)
	} else {
		finalSystemInstruction = ""
	}

	return req, nil
}

/*
printLocAIModelConfig prints relevant parts of the local AI model configuration to the console.
*/
func printLocAIModelConfig(locAIModelConfig openai.ChatCompletionRequest, terminalWidth int) {
	fmt.Printf("\nLocAI model configuration (excerpt):\n")
	if finalSystemInstruction != "" {
		fmt.Printf("  SystemInstruction : %v\n", wrapString(finalSystemInstruction, terminalWidth, 22))
	}
	fmt.Printf("  CandidateCount    : %v\n", locAIModelConfig.N)
	if locAIModelConfig.MaxTokens > 0 {
		fmt.Printf("  MaxOutputTokens   : %v\n", locAIModelConfig.MaxTokens)
	}
}

/*
showCompactConfiguration shows a very compact overview of the most important parameters.
*/
func showCompactConfiguration(modelInfo *openai.Model, _ openai.ChatCompletionRequest) {
	fmt.Printf("\n--- %s %s ---------------------------------------------------\n", progName, progVersion)
	fmt.Printf("Model  : %s\n", modelInfo.ID)

	// Config
	configParts := []string{fmt.Sprintf("%d Candidate(s)", *progConfig.LocAICandidateCount)}
	fmt.Printf("Config : %s\n", strings.Join(configParts, ", "))

	// Context (Files, Cache, RAG)
	okCount := 0
	warnCount := 0
	errorCount := 0
	for _, f := range filesToHandle {
		switch f.State {
		case "ok":
			okCount++
		case "warn":
			warnCount++
		case "error":
			errorCount++
		}
	}

	if len(filesToHandle) > 0 {
		loadedCount := okCount + warnCount
		infoPart := fmt.Sprintf("Files  : %d local files loaded", loadedCount)

		var details []string
		if warnCount > 0 {
			details = append(details, fmt.Sprintf("%d %s", warnCount, pluralize(warnCount, "warning")))
		}
		if errorCount > 0 {
			details = append(details, fmt.Sprintf("%d %s", errorCount, pluralize(errorCount, "error")))
		}

		if len(details) > 0 {
			infoPart += fmt.Sprintf(" (%s)", strings.Join(details, ", "))
		}
		fmt.Printf("%s\n", infoPart)
	}

	// Mode
	modeStr := "Non-Chat"
	if *chatmode {
		modeStr = "Chat"
	}
	if progConfig.LocAIPureResponse {
		modeStr += ", Pure-Response"
	}
	fmt.Printf("Mode   : %s\n", modeStr)

	// Output
	var outputs []string
	if progConfig.AnsiOutput {
		outputs = append(outputs, "Terminal")
	}
	if progConfig.HTMLOutput {
		outputs = append(outputs, "HTML")
	}
	if progConfig.MarkdownOutput {
		outputs = append(outputs, "Markdown")
	}
	fmt.Printf("Output : %s\n", strings.Join(outputs, ", "))

	fmt.Printf("Quit   : CTRL-C\n")
}
