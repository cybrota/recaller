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

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type HistoryConfig struct {
	EnableFuzzing bool `yaml:"enable_fuzzing"`
}

type Config struct {
	History HistoryConfig `yaml:"history"`
}

var defaultConfig = Config{
	History: HistoryConfig{
		EnableFuzzing: true,
	},
}

func LoadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return &defaultConfig, nil
	}

	configPath := filepath.Join(homeDir, ".recaller.yaml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &defaultConfig, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return &defaultConfig, nil
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return &defaultConfig, nil
	}

	return &config, nil
}

func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".recaller.yaml"), nil
}

func createDefaultConfigFile() error {
	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %v", err)
	}

	data, err := yaml.Marshal(&defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %v", err)
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

func displaySettings() {
	configPath, err := getConfigPath()
	if err != nil {
		fmt.Printf("❌ Failed to get config path: %v\n", err)
		return
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		return
	}

	configExists := true
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configExists = false
		fmt.Printf("📝 Configuration file not found. Creating default configuration...\n\n")

		if err := createDefaultConfigFile(); err != nil {
			fmt.Printf("❌ Failed to create default config file: %v\n", err)
			return
		}
		fmt.Printf("✅ Created default configuration at: %s\n\n", configPath)
	}

	fmt.Printf("🔧 Recaller Configuration Settings\n")
	fmt.Printf("═══════════════════════════════════\n\n")

	if configExists {
		fmt.Printf("📍 Config file: %s\n", configPath)
	} else {
		fmt.Printf("📍 Config file: %s (newly created)\n", configPath)
	}

	fmt.Printf("📊 Current settings:\n\n")

	fmt.Printf("🔍 %sHistory Search:%s\n", Green, Reset)

	fuzzyValue := "true"
	fuzzyDesc := "Fuzzy search (substring matching anywhere)"
	if !config.History.EnableFuzzing {
		fuzzyValue = "false"
		fuzzyDesc = "Prefix-based search (commands starting with query)"
	}

	fmt.Printf("  • %senable_fuzzing%s: %s\n", Green, Reset, fuzzyValue)
	fmt.Printf("    %s\n\n", fuzzyDesc)

	if !config.History.EnableFuzzing {
		fmt.Printf("💡 Fuzzy search is disabled. To enable it, edit %s:\n", configPath)
		fmt.Printf("   history:\n     enable_fuzzing: true\n\n")
	} else {
		fmt.Printf("💡 To use prefix-only search, edit %s:\n", configPath)
		fmt.Printf("   history:\n     enable_fuzzing: false\n\n")
	}

	fmt.Printf("📚 For more information, see: https://github.com/cybrota/recaller#search-modes\n")
}
