//go:fmt off
/*
Purpose:
- local AI prompt (locai-pro)

Description:
- Prompts local AI model and displays the response.

Releases:
  - v0.1.0 - 2025-04-21: initial release

Copyright:
- © 2026 | Klaus Tockloth

License:
- MIT License

Contact:
- klaus.tockloth@googlemail.com

Remarks:
- Migrated from gem-pro (Gemini Prompt) and Google GenAI SDK.

ToDos:
- none

Links:
- none
*/
//go:fmt on

// main package
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gohugoio/hugo-goldmark-extensions/passthrough"
	"github.com/sashabaranov/go-openai"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// general program info
var (
	progName    = strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(filepath.Base(os.Args[0])))
	progVersion = "v0.1.0"
	progDate    = "2026-04-21"
	progPurpose = "local AI prompt (locai-pro)"
	progInfo    = "Prompts local AI model and displays the response."
)

// processing timestamp
var (
	startProcessing  time.Time
	finishProcessing time.Time
)

// markdown to html parser
var markdownParser goldmark.Markdown

// FileToHandle represents all files to handle in prompt
type FileToHandle struct {
	State        string
	Filepath     string
	LastUpdate   string
	FileSize     string
	MimeType     string
	ErrorMessage string
}

// filesToHandle holds list of files to handle in prompt
var filesToHandle []FileToHandle

// finalSystemInstruction holds the complete system prompt (App + User)
var finalSystemInstruction string

// stringArray implements the flag.Value interface.
type stringArray []string

/*
String is for the output of the default value in the help text.
*/
func (s *stringArray) String() string {
	return fmt.Sprint(*s)
}

/*
Set is called each time the flag is found.
*/
func (s *stringArray) Set(value string) error {
	*s = append(*s, value)
	return nil
}

/*
Get implements the flag.Getter interface.
*/
func (s *stringArray) Get() interface{} {
	return []string(*s)
}

// command line parameters
var (
	candidates = flag.Int("candidates", 0, "Specifies the number of candidate responses the AI should generate.")
	config     = flag.String("config", progName+".yaml", "Specifies the name of the YAML configuration file.")
	// special handling for option 'filelist'
	chatmode     = flag.Bool("chatmode", false, "Enables chat mode, where the AI remembers conversation history within a session.")
	listModels   = flag.Bool("list-models", false, "Lists all available local AI models and exits.")
	outputBase   = flag.String("out", "", "Specifies the base filename for the output files.\n E.g. 'response-1' -> 'response-1.md', 'response-1.html', 'response-1.ansi'.")
	pureResponse = flag.Bool("pure-response", false, "Pure response without any boilerplate.")
	verbose      = flag.Bool("verbose", false, "Detailed output of configuration and model information.")
	sysprompt    = flag.String("sysprompt", "", "Specifies the system instruction file (overrides config).")
)
var fileLists stringArray

/*
main starts this program. It is the minimal entry point of the application.
It defers execution to runMain() to ensure all defer statements are executed properly
before exiting.
*/
func main() {
	os.Exit(runMain())
}

/*
runMain is the actual main routine. It returns an exit code which is passed to os.Exit() by main().
*/
func runMain() int {
	var err error

	// register the variables for the flags
	flag.Var(&fileLists, "filelist", "Specifies a file containing a list of files to upload (can be repeated).\n"+
		"Entries are one filename per line. Empty lines and comments (# or //) are ignored.")

	flag.Usage = printUsage
	flag.Parse()

	// track which flags were actually set by the user
	setFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	// close PDF pool
	defer closePdfiumPool()

	if *verbose {
		fmt.Printf("\nProgram:\n")
		fmt.Printf("  Name    : %s\n", progName)
		fmt.Printf("  Release : %s - %s\n", progVersion, progDate)
		fmt.Printf("  Purpose : %s\n", progPurpose)
		fmt.Printf("  Info    : %s\n", progInfo)
	}

	if !fileExists(*config) {
		if err := writeConfig(); err != nil {
			fmt.Println(err)
			return 1
		}
	}
	if !fileExists("README.md") {
		if err := writeReadme(); err != nil {
			fmt.Println(err)
			return 1
		}
	}
	if !fileExists("locai-pro.png") {
		if err := writeLocAIProPng(); err != nil {
			fmt.Println(err)
			return 1
		}
	}
	if !fileExists("system-instruction.txt") {
		if err := writeSystemInstruction(); err != nil {
			fmt.Println(err)
			return 1
		}
	}

	// 'assets' in current directory (to render current HTML file in current directory)
	directory := "./assets"
	if !dirExists(directory) {
		err = os.Mkdir(directory, 0750)
		if err != nil && !os.IsExist(err) {
			fmt.Printf("error [%v] at os.Mkdir()\n", err)
			return 1
		}
		if err := writeAssets("."); err != nil {
			fmt.Println(err)
			return 1
		}
	}

	if !fileExists("./prompt-input.html") {
		if err := writePromptInput(); err != nil {
			fmt.Println(err)
			return 1
		}
	}

	err = loadConfiguration(*config)
	if err != nil {
		fmt.Printf("error [%v] loading configuration\n", err)
		return 1
	}

	// handle custom output base filename
	if *outputBase != "" {
		progConfig.MarkdownPromptResponseFile = *outputBase + ".md"
		progConfig.HTMLPromptResponseFile = *outputBase + ".html"
		progConfig.AnsiPromptResponseFile = *outputBase + ".ansi"
	}

	// set local AI model
	if progConfig.LocAIModel == "" {
		fmt.Printf("empty local AI model not allowed\n")
		return 1
	}

	// build list of files given via command line
	filesToHandle = buildGivenFiles(flag.Args(), fileLists)

	// shows files given via command line
	if *verbose {
		fmt.Printf("\nFiles given via command line:\n")
		if len(filesToHandle) == 0 {
			fmt.Printf("  none\n")
		} else {
			for _, fileToHandle := range filesToHandle {
				if fileToHandle.State != "error" {
					// add replacement MIME type (e.g. 'text/x-perl -> text/plain')
					mimeType := fileToHandle.MimeType
					fmt.Printf("  %-5s %s (%s, %s, %s)\n",
						fileToHandle.State, fileToHandle.Filepath, fileToHandle.LastUpdate, fileToHandle.FileSize, mimeType)
				} else {
					fmt.Printf("  %-5s %s %s\n",
						fileToHandle.State, fileToHandle.Filepath, fileToHandle.ErrorMessage)
				}
			}
		}
	}

	// handle standalone actions
	if *listModels {
		showAvailableLocAIModels()
		return 0
	}

	// initialize this program
	err = initializeProgram()
	if err != nil {
		fmt.Printf("%v\n", err)
		return 1
	}

	// overwrite YAML config values with cli parameters
	overwriteConfigValues(setFlags)

	// configure Markdown Passthrough Extension
	passthroughExt := passthrough.New(passthrough.Config{
		InlineDelimiters: []passthrough.Delimiters{
			{Open: "$", Close: "$"},
			{Open: `\(`, Close: `\)`},
		},
		BlockDelimiters: []passthrough.Delimiters{
			{Open: "$$", Close: "$$"},
			{Open: `\[`, Close: `\]`},
		},
	})

	// create markdown parser (WithUnsafe() ensures to render potentially dangerous links like "file:///Users/...")
	markdownParser = goldmark.New(
		goldmark.WithExtensions(extension.GFM, &TargetBlankExtension{}, passthroughExt),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	openAIConfig := openai.DefaultConfig(progConfig.LocAIAPIKey)
	openAIConfig.BaseURL = progConfig.LocAIURL
	client := openai.NewClientWithConfig(openAIConfig)

	// get local AI model information via ListModels
	modelsList, err := client.ListModels(ctx)
	if err != nil {
		fmt.Printf("error [%v] getting AI model information via ListModels\n", err)
		return 1
	}

	var locAIModelInfo *openai.Model
	for _, m := range modelsList.Models {
		if m.ID == progConfig.LocAIModel {
			locAIModelInfo = &m
			break
		}
	}
	if locAIModelInfo == nil {
		if len(modelsList.Models) > 0 {
			locAIModelInfo = &modelsList.Models[0]
			progConfig.LocAIModel = locAIModelInfo.ID
		} else {
			locAIModelInfo = &openai.Model{ID: progConfig.LocAIModel}
		}
	}

	// generate local AI model configuration
	locAIModelConfig, err := generateOpenAIModelConfig()
	if err != nil {
		fmt.Printf("%v\n", err)
		return 1
	}

	// show start/config parameter
	if *verbose {
		showConfiguration()
		printLocAIModelInfo(locAIModelInfo)
		printLocAIModelConfig(locAIModelConfig, progConfig.AnsiOutputLineLength)
	} else {
		showCompactConfiguration(locAIModelInfo, locAIModelConfig)
	}

	// define prompt channel
	promptChannel := make(chan string)

	// set up signal handling for shutdown (e.g. Ctrl-C)
	shutdownTrigger := make(chan os.Signal, 1)
	signal.Notify(shutdownTrigger, syscall.SIGINT)  // kill -SIGINT pid -> interrupt
	signal.Notify(shutdownTrigger, syscall.SIGTERM) // kill -SIGTERM pid -> terminated

	if *verbose {
		fmt.Printf("\nOperation mode:\n")
		if *chatmode {
			fmt.Printf("  Running in chat mode.\n")
		} else {
			fmt.Printf("  Running in non-chat mode.\n")
		}
		if progConfig.LocAIPureResponse {
			fmt.Printf("  Running in pure-response mode.\n")
		}

		fmt.Printf("\nProgram termination:\n")
		fmt.Printf("  Press CTRL-C to terminate this program.\n\n")
	}

	// start graceful shutdown handler
	go handleShutdown(shutdownTrigger, cancel)

	// check if input is piped
	isPiped := isInputPiped()
	inputPossibilities := []string{}

	if isPiped {
		// pipe mode: read strictly from Stdin until EOF
		go readPromptFromPipe(promptChannel)
		inputPossibilities = append(inputPossibilities, "Pipe")
	} else {
		// interactive mode: start configured readers (Terminal, File, Localhost)
		inputPossibilities = startInputReaders(promptChannel, progConfig)
	}

	chatNumber := 1
	var chatMessages []openai.ChatCompletionMessage

	// start main loop: Prompt local AI
	// --------------------------------
	var resp openai.ChatCompletionResponse
	var respErr error
	for {
		if !isPiped {
			fmt.Printf("Waiting for input from %s ...\n", strings.Join(inputPossibilities, ", "))
		} else {
			fmt.Printf("Processing piped input ...\n")
		}

		// read prompt from channel or handle shutdown
		var prompt string
		select {
		case <-ctx.Done():
			return 0
		case p, ok := <-promptChannel:
			if !ok {
				return 0
			}
			prompt = strings.TrimSpace(p)
		}

		now := time.Now()
		if progConfig.NotifyPrompt {
			err = runCommand(progConfig.NotifyPromptApplication)
			if err != nil {
				fmt.Printf("error [%v] at runCommand()\n", err)
			}
		}

		var currentMessages []openai.ChatCompletionMessage

		// build prompt parts (filedata, text prompt) for non-chat mode
		if !*chatmode {
			if finalSystemInstruction != "" {
				currentMessages = append(currentMessages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleSystem,
					Content: finalSystemInstruction,
				})
			}

			var hasImages bool
			for _, fileToHandle := range filesToHandle {
				if fileToHandle.State != "error" && strings.HasPrefix(fileToHandle.MimeType, "image/") {
					hasImages = true
					break
				}
			}

			if hasImages {
				var userParts []openai.ChatMessagePart
				for _, fileToHandle := range filesToHandle {
					if fileToHandle.State == "error" {
						continue
					}
					part, err := convertFileToMessagePart(fileToHandle.Filepath)
					if err != nil {
						fmt.Printf("error [%v] converting file to message part\n", err)
						continue
					}
					userParts = append(userParts, part)
				}
				// add text prompt
				userParts = append(userParts, openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeText,
					Text: prompt,
				})
				currentMessages = append(currentMessages, openai.ChatCompletionMessage{
					Role:         openai.ChatMessageRoleUser,
					MultiContent: userParts,
				})
			} else {
				// Text-only handling (legacy compatible)
				for _, fileToHandle := range filesToHandle {
					if fileToHandle.State == "error" {
						continue
					}
					part, err := convertFileToMessagePart(fileToHandle.Filepath)
					if err != nil {
						fmt.Printf("error [%v] converting file to message part\n", err)
						continue
					}
					currentMessages = append(currentMessages, openai.ChatCompletionMessage{
						Role:    openai.ChatMessageRoleUser,
						Content: part.Text,
					})
				}
				// add text prompt
				currentMessages = append(currentMessages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				})
			}
		}

		// build prompt parts for chat mode
		if *chatmode {
			if chatNumber == 1 {
				if finalSystemInstruction != "" {
					chatMessages = append(chatMessages, openai.ChatCompletionMessage{
						Role:    openai.ChatMessageRoleSystem,
						Content: finalSystemInstruction,
					})
				}

				var hasImages bool
				for _, fileToHandle := range filesToHandle {
					if fileToHandle.State != "error" && strings.HasPrefix(fileToHandle.MimeType, "image/") {
						hasImages = true
						break
					}
				}

				if hasImages {
					var userParts []openai.ChatMessagePart
					for _, fileToHandle := range filesToHandle {
						if fileToHandle.State == "error" {
							continue
						}
						part, err := convertFileToMessagePart(fileToHandle.Filepath)
						if err != nil {
							fmt.Printf("error [%v] converting file to message part\n", err)
							continue
						}
						userParts = append(userParts, part)
					}
					userParts = append(userParts, openai.ChatMessagePart{
						Type: openai.ChatMessagePartTypeText,
						Text: prompt,
					})
					chatMessages = append(chatMessages, openai.ChatCompletionMessage{
						Role:         openai.ChatMessageRoleUser,
						MultiContent: userParts,
					})
				} else {
					for _, fileToHandle := range filesToHandle {
						if fileToHandle.State == "error" {
							continue
						}
						part, err := convertFileToMessagePart(fileToHandle.Filepath)
						if err != nil {
							fmt.Printf("error [%v] converting file to message part\n", err)
							continue
						}
						chatMessages = append(chatMessages, openai.ChatCompletionMessage{
							Role:    openai.ChatMessageRoleUser,
							Content: part.Text,
						})
					}
					chatMessages = append(chatMessages, openai.ChatCompletionMessage{
						Role:    openai.ChatMessageRoleUser,
						Content: prompt,
					})
				}
			} else {
				// subsequent messages in chat mode are always text
				chatMessages = append(chatMessages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				})
			}
			currentMessages = chatMessages
		}

		if *chatmode {
			fmt.Printf("%02d:%02d:%02d: Processing prompt in chat mode ...\n", now.Hour(), now.Minute(), now.Second())
		} else {
			fmt.Printf("%02d:%02d:%02d: Processing prompt in non-chat mode ...\n", now.Hour(), now.Minute(), now.Second())
		}

		processPrompt(prompt, *chatmode, chatNumber)

		dumpDataToFile(os.O_TRUNC|os.O_WRONLY, "local AI model config", locAIModelConfig)
		dumpDataToFile(os.O_APPEND|os.O_CREATE|os.O_WRONLY, "local AI prompt contents", currentMessages)

		// generate content
		startProcessing = time.Now()

		req := locAIModelConfig
		req.Messages = currentMessages
		resp, respErr = client.CreateChatCompletion(ctx, req)

		finishProcessing = time.Now()

		dumpDataToFile(os.O_APPEND|os.O_CREATE|os.O_WRONLY, "local AI response", resp)
		dumpDataToFile(os.O_APPEND|os.O_CREATE|os.O_WRONLY, "local AI error", respErr)

		if *chatmode && respErr == nil && len(resp.Choices) > 0 {
			chatMessages = append(chatMessages, resp.Choices[0].Message)
		}

		// trigger response notification
		if progConfig.NotifyResponse {
			err = runCommand(progConfig.NotifyResponseApplication)
			if err != nil {
				fmt.Printf("error [%v] at runCommand()\n", err)
			}
		}

		// handle response
		handleResponse(&resp, respErr, prompt)

		// If input was piped, we are in "One-Shot" mod: process one prompt, get one response, and exit.
		if isPiped {
			return 0
		}

		// increase chat number
		if *chatmode {
			chatNumber++
		}
	}
}

/*
overwriteConfigValues overwrites configuration values in `progConfig` with values provided via command-line flags.
It updates the `progConfig` struct with values from command-line flags, allowing users to override settings from
the YAML configuration file.
*/
func overwriteConfigValues(setFlags map[string]bool) {
	if setFlags["candidates"] {
		val := int32(*candidates)
		progConfig.LocAICandidateCount = &val
	}
	if setFlags["pure-response"] {
		progConfig.LocAIPureResponse = *pureResponse
	}
	if setFlags["sysprompt"] {
		progConfig.SystemInstructionFile = *sysprompt
	}
}

/*
handleResponse processes the response received from the local AI model. It manages the AI response, including
error handling, output formatting, saving history, and triggering output applications for different formats
like Markdown and HTML.
*/
func handleResponse(resp *openai.ChatCompletionResponse, respErr error, prompt string) {
	now := finishProcessing
	fmt.Printf("%02d:%02d:%02d: Processing response ...\n", now.Hour(), now.Minute(), now.Second())
	switch {
	case respErr != nil:
		processError(respErr)
	case resp != nil:
		if progConfig.LocAIPureResponse {
			processPureResponse(resp)
		} else {
			processResponse(resp)
		}
	default:
		unknownErr := fmt.Errorf("unexpected state: received neither a response nor an error from API")
		processError(unknownErr)
	}

	// extract slug (as part of the filename)
	var slug string
	if respErr != nil {
		slug = "error-response"
	} else {
		fullText := ""
		if len(resp.Choices) > 0 {
			fullText = getCandidateText(resp.Choices[0])
		}
		_, slug = extractAndCleanSlug(fullText)
	}

	// print prompt and response to terminal
	if progConfig.AnsiOutput {
		printPromptResponseToTerminal()
	}

	// copy ansi file to history
	if progConfig.AnsiHistory {
		ansiDestinationFile := buildDestinationFilename(now, slug, "ansi")
		ansiDestinationPathFile := filepath.Join(progConfig.AnsiHistoryDirectory, ansiDestinationFile)
		copyFile(progConfig.AnsiPromptResponseFile, ansiDestinationPathFile)
	}

	// markdown prompt and response file: nothing to do
	commandLine := fmt.Sprintf(progConfig.MarkdownOutputApplication, progConfig.MarkdownPromptResponseFile)

	// copy markdown file to history
	if progConfig.MarkdownHistory {
		markdownDestinationFile := buildDestinationFilename(now, slug, "md")
		markdownDestinationPathFile := filepath.Join(progConfig.MarkdownHistoryDirectory, markdownDestinationFile)
		copyFile(progConfig.MarkdownPromptResponseFile, markdownDestinationPathFile)
		commandLine = fmt.Sprintf(progConfig.MarkdownOutputApplication, "\""+markdownDestinationPathFile+"\"")
	}

	// open markdown document in application
	if progConfig.MarkdownOutput {
		err := runCommand(commandLine)
		if err != nil {
			fmt.Printf("error [%v] at runCommand()\n", err)
		}
	}

	// build prompt and response html page
	commandLine = fmt.Sprintf(progConfig.HTMLOutputApplication, progConfig.HTMLPromptResponseFile)
	_ = buildHTMLPage(prompt, progConfig.HTMLPromptResponseFile, progConfig.HTMLPromptResponseFile)

	// copy html file to history
	if progConfig.HTMLHistory {
		htmlDestinationFile := buildDestinationFilename(now, slug, "html")
		htmlDestinationPathFile := filepath.Join(progConfig.HTMLHistoryDirectory, htmlDestinationFile)
		copyFile(progConfig.HTMLPromptResponseFile, htmlDestinationPathFile)
		commandLine = fmt.Sprintf(progConfig.HTMLOutputApplication, "\""+htmlDestinationPathFile+"\"")
	}

	// open html page in application
	if progConfig.HTMLOutput {
		err := runCommand(commandLine)
		if err != nil {
			fmt.Printf("error [%v] at runCommand()\n", err)
		}
	}
}

/*
printLocAIModelInfo prints detailed information about a local AI model to the console.
*/
func printLocAIModelInfo(locAIModelInfo *openai.Model) {
	fmt.Printf("\nLocAI model information:\n")
	fmt.Printf("  ID                : %v\n", locAIModelInfo.ID)
	fmt.Printf("  Created           : %v\n", locAIModelInfo.CreatedAt)
	fmt.Printf("  OwnedBy           : %v\n", locAIModelInfo.OwnedBy)
}

/*
handleShutdown handles program termination signals (SIGINT and SIGTERM). It listens for shutdown signals
and performs a graceful program exit when a signal is received, ensuring resources are properly released.
*/
func handleShutdown(shutdownTrigger chan os.Signal, cancel context.CancelFunc) {
	<-shutdownTrigger
	fmt.Printf("\nShutdown signal received. Exiting gracefully ...\n")
	cancel()
}

/*
startInputReaders initializes and starts input reader goroutines based on the program configuration. It sets
up and starts goroutines for reading prompts from different input sources like terminal, file, or localhost,
based on the configuration.
*/
func startInputReaders(promptChannel chan string, config ProgConfig) []string {
	inputPossibilities := []string{}

	// input from keyboard
	if config.InputFromTerminal {
		go readPromptFromKeyboard(promptChannel)
		inputPossibilities = append(inputPossibilities, "Terminal")
	}

	// input from file
	if config.InputFromFile {
		if !fileExists(config.InputFile) {
			file, err := os.Create(config.InputFile)
			if err != nil {
				fmt.Printf("error [%v] creating input prompt text file\n", err)
				return inputPossibilities
			}
			_ = file.Close()
		}
		go readPromptFromFile(config.InputFile, promptChannel)
		inputPossibilities = append(inputPossibilities, "File")
	}

	// input from localhost
	if config.InputFromLocalhost {
		addr := fmt.Sprintf("localhost:%d", config.InputLocalhostPort)
		go func() {
			http.HandleFunc("/", readPromptFromLocalhost(promptChannel))
			err := http.ListenAndServe(addr, nil)
			if err != nil {
				fmt.Printf("error [%v] starting internal webserver\n", err)
				return
			}
		}()
		inputPossibilities = append(inputPossibilities, addr)
	}

	return inputPossibilities
}

/*
isInputPiped verifies if input pipe is connected.
*/
func isInputPiped() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}

/*
buildGivenFiles builds a list of files provided via command-line (list, args). It processes file paths from
command-line arguments and a file list, checks their state, and prepares a list of FileToHandle structures
for further processing.
*/
func buildGivenFiles(args []string, filelists []string) []FileToHandle {
	var filesFromList []string

	for _, listFile := range filelists {
		lines, err := slurpFile(listFile)
		if err != nil {
			fmt.Printf("error [%v] reading list of files from [%s]\n", err, listFile)
			continue
		}
		filesFromList = append(filesFromList, lines...)
	}

	files := filesFromList
	files = append(files, args...)

	for _, file := range files {
		fileInfo, err := os.Stat(file)
		if err != nil {
			filesToHandle = append(filesToHandle, FileToHandle{
				Filepath:     file,
				State:        "error",
				ErrorMessage: fmt.Sprintf("error [%v] at os.Stat()", err),
			})
			continue
		}

		mimeType, err := getFileMimeType(file)
		if err != nil {
			filesToHandle = append(filesToHandle, FileToHandle{
				Filepath:     file,
				State:        "error",
				ErrorMessage: fmt.Sprintf("error [%v] at getFileMimeType()", err),
			})
			continue
		}

		cleanMime := strings.Split(mimeType, ";")[0]

		// PDF interception
		if cleanMime == "application/pdf" {
			pdfBase := filepath.Base(file)
			// create specific temp directory for this PDF (prevents overlaps)
			outDir := filepath.Join(".", ".tmp-pdf-images", pdfBase)

			// delete existing directory before generation
			_ = os.RemoveAll(outDir)

			convertedCount, err := convertPDFToImages(file, outDir, 150, 90)
			if err != nil {
				filesToHandle = append(filesToHandle, FileToHandle{
					Filepath:     file,
					State:        "error",
					ErrorMessage: fmt.Sprintf("PDF conversion error: %v", err),
				})
				continue
			}

			// load the generated images as new FileToHandle objects (ignoring the original PDF)
			for i := 1; i <= convertedCount; i++ {
				imgPath := filepath.Join(outDir, fmt.Sprintf("%s.%03d.jpg", pdfBase, i))
				imgInfo, imgErr := os.Stat(imgPath)
				if imgErr != nil {
					continue
				}

				filesToHandle = append(filesToHandle, FileToHandle{
					Filepath:   imgPath,
					State:      "ok",
					FileSize:   fmt.Sprintf("%.1f KiB", float64(imgInfo.Size())/1024.0),
					LastUpdate: imgInfo.ModTime().Format("20060102-150405"),
					MimeType:   "image/jpeg",
				})
			}
			continue // iterate to next file
		}

		isText := false
		if strings.HasPrefix(cleanMime, "text/") {
			isText = true
		} else {
			switch cleanMime {
			case "application/json", "application/xml", "application/javascript",
				"application/x-yaml", "application/toml", "application/sql",
				"application/x-sh", "application/x-shellscript", "application/graphql":
				isText = true
			}
		}

		isImage := strings.HasPrefix(cleanMime, "image/")

		info := "ok"
		errMsg := ""
		if !isText && !isImage {
			info = "error"
			errMsg = fmt.Sprintf("binary files (MIME: %s) are not supported", cleanMime)
		}

		filesToHandle = append(filesToHandle, FileToHandle{
			Filepath:     file,
			State:        info,
			FileSize:     fmt.Sprintf("%.1f KiB", float64(fileInfo.Size())/1024.0),
			LastUpdate:   fileInfo.ModTime().Format("20060102-150405"),
			MimeType:     mimeType,
			ErrorMessage: errMsg,
		})
	}

	return filesToHandle
}
