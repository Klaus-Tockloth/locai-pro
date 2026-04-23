package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

/*
printPromptResponseToTerminal prints the content of the ANSI prompt/response file to the standard output (terminal).
It reads the content from the ANSI formatted prompt / response file and writes it directly to the standard output,
displaying colored text in the terminal.
*/
func printPromptResponseToTerminal() {
	data, err := os.ReadFile(progConfig.AnsiPromptResponseFile)
	if err != nil {
		fmt.Printf("error [%v] at os.ReadFile()\n", err)
		return
	}
	_, _ = os.Stdout.Write(data)
}

/*
processPrompt processes the user prompt and prepares it for different output formats (Markdown, ANSI, HTML).
It takes a user prompt, formats it into Markdown, ANSI, and HTML, including system instructions and referenced
files, and saves these formats to respective files.
*/
func processPrompt(prompt string, chatmode bool, chatNumber int) {
	// If pure response is requested, do not write prompt to output files.
	// But ensure files are empty/truncated so they don't contain old data.
	if progConfig.LocAIPureResponse {
		_ = os.WriteFile(progConfig.MarkdownPromptResponseFile, []byte(""), 0600)
		_ = os.WriteFile(progConfig.AnsiPromptResponseFile, []byte(""), 0600)
		_ = os.WriteFile(progConfig.HTMLPromptResponseFile, []byte(""), 0600)
		return
	}

	var promptString strings.Builder

	// text part of prompt (also included in contents)
	promptString.WriteString("***\n")
	if chatmode {
		if chatNumber == 1 {
			promptString.WriteString("**Prompt to LocAI (initial chat #1):**\n\n")
		} else {
			fmt.Fprintf(&promptString, "**Prompt to LocAI (refinement chat #%d):**\n\n", chatNumber)
		}
	} else {
		promptString.WriteString("**Prompt to LocAI:**\n\n")
	}
	promptString.WriteString("<!-- PROMPT_USER_START -->\n")
	promptString.WriteString("```plaintext\n")
	promptString.WriteString(prompt)
	promptString.WriteString("\n```\n")
	promptString.WriteString("<!-- PROMPT_USER_END -->\n")

	// system instructions part of prompt (not included in contents, but important)
	if finalSystemInstruction != "" {
		promptString.WriteString("\n<!-- PROMPT_SYSTEM_START -->\n")
		promptString.WriteString("```plaintext\n")
		promptString.WriteString(finalSystemInstruction)
		promptString.WriteString("\n```\n")
		promptString.WriteString("<!-- PROMPT_SYSTEM_END -->\n")
	}

	if (chatmode && chatNumber == 1) || !chatmode {
		if len(filesToHandle) > 0 {
			promptString.WriteString("\n<!-- PROMPT_FILES_START -->\n")
			promptString.WriteString("```plaintext\n")
			for _, fileToUpload := range filesToHandle {
				if fileToUpload.State != "error" {
					// add replacement MIME type (e.g. 'text/x-perl -> text/plain')
					mimeType := fileToUpload.MimeType
					fmt.Fprintf(&promptString, "%-5s %s (%s, %s, %s)\n",
						fileToUpload.State, fileToUpload.Filepath, fileToUpload.LastUpdate, fileToUpload.FileSize, mimeType)
				} else {
					fmt.Fprintf(&promptString, "%-5s %s %s\n",
						fileToUpload.State, fileToUpload.Filepath, fileToUpload.ErrorMessage)
				}
			}
			promptString.WriteString("```\n")
			promptString.WriteString("<!-- PROMPT_FILES_END -->\n")
		}
	}
	promptString.WriteString("\n***\n")

	rawPrompt := promptString.String()

	// 1. prepare Markdown for direct file saving (and ANSI rendering)
	markdownForFileAndAnsi := rawPrompt
	markdownForFileAndAnsi = strings.ReplaceAll(markdownForFileAndAnsi, "<!-- PROMPT_USER_START -->", "**User-Prompt:**\n")
	markdownForFileAndAnsi = strings.ReplaceAll(markdownForFileAndAnsi, "<!-- PROMPT_USER_END -->", "")
	markdownForFileAndAnsi = strings.ReplaceAll(markdownForFileAndAnsi, "<!-- PROMPT_SYSTEM_START -->", "**System-Prompt:**\n")
	markdownForFileAndAnsi = strings.ReplaceAll(markdownForFileAndAnsi, "<!-- PROMPT_SYSTEM_END -->", "")
	markdownForFileAndAnsi = strings.ReplaceAll(markdownForFileAndAnsi, "<!-- PROMPT_FILES_START -->", "**Files:**\n")
	markdownForFileAndAnsi = strings.ReplaceAll(markdownForFileAndAnsi, "<!-- PROMPT_FILES_END -->", "")

	// write prompt to current markdown request/response file
	err := os.WriteFile(progConfig.MarkdownPromptResponseFile, []byte(markdownForFileAndAnsi), 0600)
	if err != nil {
		fmt.Printf("error [%v] at os.WriteFile()\n", err)
		return
	}

	// render prompt as ansi
	ansiData := markdownForFileAndAnsi
	if progConfig.AnsiRendering {
		ansiData = renderMarkdown2Ansi(markdownForFileAndAnsi)
	}

	// write prompt to current ansi request/response file
	err = os.WriteFile(progConfig.AnsiPromptResponseFile, []byte(ansiData), 0600)
	if err != nil {
		fmt.Printf("error [%v] at os.WriteFile()\n", err)
		return
	}

	// render prompt as html
	htmlData := rawPrompt
	if progConfig.HTMLRendering {
		htmlData = renderMarkdown2HTML(rawPrompt)
	}

	// write prompt to current html request/response file
	err = os.WriteFile(progConfig.HTMLPromptResponseFile, []byte(htmlData), 0600)
	if err != nil {
		fmt.Printf("error [%v] at os.WriteFile()\n", err)
		return
	}
}

/*
getCandidateText extracts the text content from a choice, explicitly
including native reasoning content if provided by the API.
*/
func getCandidateText(choice openai.ChatCompletionChoice) string {
	var sb strings.Builder

	// 1. Check for native ReasoningContent (OpenAI API spec / llama-server)
	if choice.Message.ReasoningContent != "" {
		sb.WriteString("<!-- AI_THOUGHT_SUMMARY_START -->\n<!-- AI_THOUGHT_SUMMARY_END -->\n<!-- AI_THOUGHT_CONTENT_START -->\n")
		sb.WriteString(choice.Message.ReasoningContent)
		sb.WriteString("\n<!-- AI_THOUGHT_CONTENT_END -->\n\n")
	}

	text := choice.Message.Content

	// 2. Fallback: Parse inline tags for models/servers that still output thoughts within the main content
	// DeepSeek tags: <think> ... </think>
	text = strings.ReplaceAll(text, "<think>", "<!-- AI_THOUGHT_SUMMARY_START -->\n<!-- AI_THOUGHT_SUMMARY_END -->\n<!-- AI_THOUGHT_CONTENT_START -->\n")
	text = strings.ReplaceAll(text, "</think>", "\n<!-- AI_THOUGHT_CONTENT_END -->\n\n")

	// Gemma 4 tags: <|channel> ... <channel|>
	text = strings.ReplaceAll(text, "<|channel>thought", "<!-- AI_THOUGHT_SUMMARY_START -->\n<!-- AI_THOUGHT_SUMMARY_END -->\n<!-- AI_THOUGHT_CONTENT_START -->")
	text = strings.ReplaceAll(text, "<channel|>", "\n<!-- AI_THOUGHT_CONTENT_END -->\n\n")

	text = removeSpacesBetweenNewlineAndCodeblock(text)
	sb.WriteString(text)
	sb.WriteString("\n")

	finalText := sb.String()

	// If absolutely no content was generated
	if strings.TrimSpace(finalText) == "" {
		return "No content available in this candidate.\n"
	}

	return finalText
}

/*
processPureResponse processes the local AI model's response and formats it for output.
*/
func processPureResponse(resp *openai.ChatCompletionResponse) {
	var responseString strings.Builder

	// print response candidate(s)
	for _, choice := range resp.Choices {
		// get text content, explicitly excluding thoughts
		responseString.WriteString(getCandidateText(choice))

		// show why the model stopped generating tokens (content)
		if choice.FinishReason != openai.FinishReasonStop && choice.FinishReason != "" {
			responseString.WriteString("\n***\n")
			fmt.Fprintf(&responseString, "Model stopped generating tokens (content) with reason [%s].\n", choice.FinishReason)
		}
	}

	// append response string to request/response files
	appendResponseString(responseString)
}

/*
processResponse processes the local AI model's response and formats it for output.
*/
func processResponse(resp *openai.ChatCompletionResponse) {
	var responseString strings.Builder

	// print response candidate(s)
	for i, choice := range resp.Choices {
		if len(resp.Choices) > 1 {
			fmt.Fprintf(&responseString, "**Response from LocAI (Candidate #%d):**\n\n", (i + 1))
		} else {
			responseString.WriteString("**Response from LocAI:**\n\n")
		}

		// get text content, including thoughts based on config
		responseString.WriteString(getCandidateText(choice))

		// show why the model stopped generating tokens
		if choice.FinishReason != openai.FinishReasonStop && choice.FinishReason != "" {
			responseString.WriteString("\n***\n")
			fmt.Fprintf(&responseString, "Model stopped generating tokens (content) with reason [%s].\n", choice.FinishReason)
		}

		responseString.WriteString("\n***\n")
	}

	// print response metadata
	responseString.WriteString("```plaintext\n")
	fmt.Fprintf(&responseString, "AI model   : %v\n", resp.Model)

	slug := "unknown-content"
	if len(resp.Choices) > 0 {
		_, extractedSlug := extractAndCleanSlug(getCandidateText(resp.Choices[0]))
		if extractedSlug != "" {
			slug = extractedSlug
		}
	}
	fmt.Fprintf(&responseString, "Slug       : %v\n", slug)

	fmt.Fprintf(&responseString, "Generated  : %v\n", finishProcessing.Format(time.RFC850))

	duration := finishProcessing.Sub(startProcessing)
	fmt.Fprintf(&responseString, "Processing : %.1f secs for %d %s\n", duration.Seconds(),
		len(resp.Choices), pluralize(len(resp.Choices), "candidate"))

	// token usage evaluation
	if resp.Usage.TotalTokens > 0 {
		fmt.Fprintf(&responseString, "Tokens     : %d (Total)\n", resp.Usage.TotalTokens)
		fmt.Fprintf(&responseString, "  Input    : %d (Prompt: %d)\n", resp.Usage.PromptTokens, resp.Usage.PromptTokens)

		reasoningTokens := 0
		if resp.Usage.CompletionTokensDetails != nil {
			reasoningTokens = resp.Usage.CompletionTokensDetails.ReasoningTokens
		}
		candidateTokens := resp.Usage.CompletionTokens - reasoningTokens

		if reasoningTokens > 0 {
			fmt.Fprintf(&responseString, "  Output   : %d (Candidates: %d, Thoughts: %d)\n", resp.Usage.CompletionTokens, candidateTokens, reasoningTokens)
		} else {
			fmt.Fprintf(&responseString, "  Output   : %d (Candidates: %d)\n", resp.Usage.CompletionTokens, candidateTokens)
		}
	}

	responseString.WriteString("```\n")
	responseString.WriteString("\n***\n")

	// append response string to request/response files
	appendResponseString(responseString)
}

/*
processError processes errors received from the local AI model. It handles error responses from the local AI
model, formats the error message in Markdown, and prepares it for output, including metadata about the error.
*/
func processError(err error) {
	var responseString strings.Builder

	// handle error response
	responseString.WriteString("**Error Response from LocAI:**\n\n")
	responseString.WriteString("```\n")
	responseString.WriteString(err.Error())
	responseString.WriteString("\n")

	responseString.WriteString("```\n")
	responseString.WriteString("\n***\n")

	// print response metadata
	responseString.WriteString("```plaintext\n")
	if err == nil {
		fmt.Fprintf(&responseString, "AI model   : %v\n", progConfig.LocAIModel)
	}

	fmt.Fprintf(&responseString, "Slug       : error-response\n")

	fmt.Fprintf(&responseString, "Generated  : %v\n", finishProcessing.Format(time.RFC850))

	duration := finishProcessing.Sub(startProcessing)
	fmt.Fprintf(&responseString, "Processing : %.1f secs resulting in error\n", duration.Seconds())

	responseString.WriteString("```\n")
	responseString.WriteString("\n***\n")

	// append response string to request/response files
	appendResponseString(responseString)
}

/*
appendResponseString appends a given response string (which can be a successful response or an error message)
to the current request / response files in Markdown, ANSI, and HTML formats.
*/
func appendResponseString(responseString strings.Builder) {
	rawMarkdown := responseString.String()

	// extraxt Metadata Slug
	cleanedContent, _ := extractAndCleanSlug(rawMarkdown)

	// cleanup Markdown
	cleanedMarkdown := cleanMarkdown(cleanedContent)

	// prepare Markdown for direct file saving (and ANSI rendering)
	// replace HTML comment tags with pure Markdown equivalents
	markdownForFileAndAnsi := cleanedMarkdown
	markdownForFileAndAnsi = strings.ReplaceAll(markdownForFileAndAnsi, "<!-- AI_THOUGHT_SUMMARY_START -->", "**Thoughts:**\n\n")
	markdownForFileAndAnsi = strings.ReplaceAll(markdownForFileAndAnsi, "<!-- AI_THOUGHT_SUMMARY_END -->", "")
	markdownForFileAndAnsi = strings.ReplaceAll(markdownForFileAndAnsi, "<!-- AI_THOUGHT_CONTENT_START -->", "")
	markdownForFileAndAnsi = strings.ReplaceAll(markdownForFileAndAnsi, "<!-- AI_THOUGHT_CONTENT_END -->", "")

	// append response string to current markdown request/response file
	currentFileMarkdown, err := os.OpenFile(progConfig.MarkdownPromptResponseFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("error [%v] at os.OpenFile() for Markdown\n", err)
		return
	}
	defer func() { _ = currentFileMarkdown.Close() }()
	_, err = fmt.Fprint(currentFileMarkdown, markdownForFileAndAnsi)
	if err != nil {
		fmt.Printf("error [%v] writing to Markdown file\n", err)
	}

	// render markdown response as ansi
	ansiData := markdownForFileAndAnsi // use the cleaned version
	if progConfig.AnsiRendering {
		ansiData = renderMarkdown2Ansi(markdownForFileAndAnsi) // pass the cleaned version
	}

	// append response string to current ansi request/response file
	currentFileAnsi, err := os.OpenFile(progConfig.AnsiPromptResponseFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("error [%v] at os.OpenFile() for ANSI\n", err)
	} else {
		defer func() { _ = currentFileAnsi.Close() }()
		_, err = fmt.Fprint(currentFileAnsi, ansiData)
		if err != nil {
			fmt.Printf("error [%v] writing to ANSI file\n", err)
		}
	}

	// render markdown response as html (using cleaned string with comments)
	htmlData := cleanedMarkdown
	if progConfig.HTMLRendering {
		htmlData = renderMarkdown2HTML(cleanedMarkdown)
	}

	// append response string to current html request/response file
	currentFileHTML, err := os.OpenFile(progConfig.HTMLPromptResponseFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("error [%v] at os.OpenFile() for HTML\n", err)
	} else {
		defer func() { _ = currentFileHTML.Close() }()
		_, err = fmt.Fprint(currentFileHTML, htmlData)
		if err != nil {
			fmt.Printf("error [%v] writing to HTML file\n", err)
		}
	}
}
