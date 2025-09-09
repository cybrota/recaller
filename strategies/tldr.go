// Copyright 2025 Naren Yellavula
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package strategies

import (
	"fmt"
	"io"
	"net/http"
)

// TldrStrategy fetches help from TLDR pages - prioritized for cleaner examples
type TldrStrategy struct{}

func (t *TldrStrategy) SupportsCommand(baseCmd string) bool {
	return true // Supports any command as it's a universal fallback
}

func (t *TldrStrategy) Priority() int {
	return 0 // Highest priority - try first for better user experience
}

func (t *TldrStrategy) GetHelp(cmdParts []string) (string, error) {
	cmd := NewCommand(cmdParts)

	baseUrl := "https://raw.githubusercontent.com/tldr-pages/tldr/refs/heads/main/pages/common"
	var fullURL string

	// Support up to 2 levels of sub-commands for TLDR
	if cmd.HasSubCommand(1) {
		subCmd := cmd.GetSubCommand(0)
		fullURL = fmt.Sprintf("%s/%s-%s.md", baseUrl, cmd.BaseCmd, subCmd)
	} else {
		fullURL = fmt.Sprintf("%s/%s.md", baseUrl, cmd.BaseCmd)
	}

	client := &http.Client{Timeout: HttpTimeout}
	resp, err := client.Get(fullURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch TLDR page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("TLDR page not found (HTTP %d)", resp.StatusCode)
	}

	limitedReader := io.LimitReader(resp.Body, MaxTldrSize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read TLDR response: %v", err)
	}

	content := string(body)
	if content != "" {
		content = "📚 TLDR Documentation:\n\n" + content
	}

	return content, nil
}
