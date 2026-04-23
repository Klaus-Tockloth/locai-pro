package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/sashabaranov/go-openai"
)

/*
convertFileToMessagePart converts a file into a ChatMessagePart for the OpenAI API.
It detects if the file is an image and encodes it as a base64 Data URL.
Otherwise, it treats it as text and structures it with headers.
*/
func convertFileToMessagePart(filepath string) (openai.ChatMessagePart, error) {
	mimeType, err := getFileMimeType(filepath)
	if err != nil {
		return openai.ChatMessagePart{}, err
	}
	data, err := os.ReadFile(filepath)
	if err != nil {
		return openai.ChatMessagePart{}, err
	}

	// encode images as base64 data URL and prepare for vision models
	if strings.HasPrefix(mimeType, "image/") {
		base64Image := base64.StdEncoding.EncodeToString(data)
		url := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image)
		return openai.ChatMessagePart{
			Type: openai.ChatMessagePartTypeImageURL,
			ImageURL: &openai.ChatMessageImageURL{
				URL: url,
			},
		}, nil
	}

	// generate a plaintext string with file headers for the local model
	fileContent := fmt.Sprintf("--- FILE: %s (MIME: %s) ---\n%s\n--- END OF FILE ---\n\n", filepath, mimeType, string(data))

	return openai.ChatMessagePart{
		Type: openai.ChatMessagePartTypeText,
		Text: fileContent,
	}, nil
}
