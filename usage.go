package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

/*
printUsage prints the program's usage instructions to the standard output.
*/
func printUsage() {
	fmt.Printf("\nUsage:\n")
	fmt.Printf("  %s [options] [files]\n", progName)

	fmt.Printf("\nExamples:\n")

	// Interactive
	fmt.Printf("  %-30s %s\n", "[Interactive Mode]", progName)

	// Piping
	fmt.Printf("  %-30s %s\n", "[Piped Input]", "cat task.txt | "+progName+" -out result")
	fmt.Printf("  %-30s %s\n", "[Pure Response]", "echo \"Hello\" | "+progName+" -pure-response")

	// Files
	fmt.Printf("  %-30s %s\n", "[Local source files]", progName+" main.go utils.go")
	fmt.Printf("  %-30s %s\n", "[Use list of files]", progName+" -filelist sources.txt")

	// Configuration
	fmt.Printf("  %-30s %s\n", "[Custom System Prompt]", progName+" -sysprompt my-instructions.txt")

	// Groups
	groups := []struct {
		name  string
		flags []string
	}{
		{"Model Selection", []string{"list-models"}},
		{"Generation Parameters", []string{"candidates", "pure-response"}},
		{"Chat & Interaction", []string{"chatmode", "verbose", "config", "filelist", "sysprompt"}},
		{"Output Control", []string{"out"}},
	}

	fmt.Printf("\nOptions:\n")
	fmt.Println("  Note: CLI flags override settings in your YAML configuration. Use '-flag=false' to disable boolean options.")

	for _, group := range groups {
		fmt.Printf("\n  [%s]\n", group.name)
		for _, flagName := range group.flags {
			f := flag.Lookup(flagName)
			if f == nil {
				continue
			}

			// Placeholder
			placeholder := ""
			if getter, ok := f.Value.(interface{ Get() interface{} }); ok {
				switch getter.Get().(type) {
				case string, *stringArray, stringArray, []string:
					placeholder = " <str>"
				case int, int32, int64:
					placeholder = " <int>"
				}
			}

			// Flag Name + Placeholder
			flagPart := fmt.Sprintf("-%s%s", f.Name, placeholder)

			// Indent Usage Text
			usageLines := strings.Split(f.Usage, "\n")
			fmt.Printf("    %-28s %s", flagPart, usageLines[0])

			// Show default if appropriate
			if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "[]" {
				fmt.Printf(" (default: %s)", f.DefValue)
			}
			fmt.Println()

			// Indent further description lines
			for _, line := range usageLines[1:] {
				fmt.Printf("    %-28s %s\n", "", strings.TrimSpace(line))
			}
		}
	}

	fmt.Printf("\nCore Concepts:\n")
	fmt.Printf("  %-30s %s\n", "[Input Channels]", "Interactive Terminal, File-Watch (prompt-input.txt), localhost:4343.")
	fmt.Printf("  %-30s %s\n", "[Terminal Inject]", "Type '<<< filename.txt' in terminal to load file content as prompt.")
	fmt.Printf("  %-30s %s\n", "[Output Formats]", "Markdown (raw), ANSI (terminal color), HTML (browser with JS features).")
	fmt.Printf("  %-30s %s\n", "[Chat Mode]", "AI remembers history. Files are sent only with the FIRST prompt.")
	fmt.Printf("  %-30s %s\n", "[Non-Chat Mode]", "Each prompt is isolated. Files are sent with EVERY prompt.")
	fmt.Printf("  %-30s %s\n", "[File Lists]", "Files passed via -filelist can contain comments (# or //)")
	fmt.Printf("  %-30s %s\n", "", "and empty lines, which will be ignored during processing.")
	fmt.Printf("  %-30s %s\n\n", "[Exit Interactive]", "Type Ctrl+C to quit.")
}

/*
showAvailableLocAIModels retrieves and displays a list of available local AI models.
*/
func showAvailableLocAIModels() {
	openAIConfig := openai.DefaultConfig(progConfig.LocAIAPIKey)
	openAIConfig.BaseURL = progConfig.LocAIURL
	client := openai.NewClientWithConfig(openAIConfig)

	// get local AI model information via ListModels
	ctx := context.Background()
	modelsList, err := client.ListModels(ctx)
	if err != nil {
		fmt.Printf("error [%v] getting AI model information via ListModels\n", err)
		return
	}

	// list all available local AI models
	fmt.Printf("\nAvailable local AI models:\n")
	for _, model := range modelsList.Models {
		fmt.Printf("- %s\n", model.ID)
	}
	fmt.Printf("\n")
}
