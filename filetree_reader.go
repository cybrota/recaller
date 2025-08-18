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
	"sort"
	"strings"
)

// ReadFilesAndDirs reads files and directories names from current folder
// Color is TermUI text-highlighting color: Ex: green
// func ReadFilesAndDirs(color string) ([]string, error) {
// 	// Read entries from the current directory
// 	entries, err := os.ReadDir(".")
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Sort the entries by their names (case-insensitive)
// 	sort.Slice(entries, func(i, j int) bool {
// 		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
// 	})

// 	var results []string
// 	// Iterate over the sorted entries
// 	for _, entry := range entries {
// 		if entry.IsDir() {
// 			// Format directories as: [dir_name](fg:green)/
// 			results = append(results, fmt.Sprintf("[%s](fg:%s)/", entry.Name(), color))
// 		} else {
// 			// For files, just use the file name with extension
// 			results = append(results, entry.Name())
// 		}
// 	}

// 	return results, nil
// }

// ReadFilesAndDirs returns file/directory names and a simple type indicator.
func ReadFilesAndDirs(color string) ([][2]string, error) {
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	var results [][2]string
	for _, entry := range entries {
		if entry.IsDir() {
			results = append(results, [2]string{fmt.Sprintf("[%s](fg:%s)/", entry.Name(), color), "Directory"})
		} else {
			results = append(results, [2]string{entry.Name(), "File"})
		}
	}
	return results, nil
}

// GroupStrings takes a slice of strings and groups every three strings into one,
// concatenating them with a space separator. If the number of strings is not a multiple of three,
// the final group will contain the remaining strings.
func GroupStrings(input []string) []string {
	var grouped []string
	for i := 0; i < len(input); i += 3 {
		end := i + 3
		if end > len(input) {
			end = len(input)
		}
		// Join the current group of strings with a space.
		group := strings.Join(input[i:end], " ")
		grouped = append(grouped, group)
	}
	return grouped
}
