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
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
)

func TestCacheHelpPageAndGetHelpPage(t *testing.T) {
	// Use the optimized cache
	c := NewOptimizedHelpCache()
	cmd := "testCommand"
	helpText := "This is help text for testCommand"

	// Initially, GetHelpPage should return an empty string for a missing command.
	if got := GetHelpPage(c, cmd); got != "" {
		t.Errorf("GetHelpPage(%q) = %q; want empty string", cmd, got)
	}

	// Cache the help text.
	CacheHelpPage(c, cmd, helpText)

	// Now, GetHelpPage should return the cached help text.
	if got := GetHelpPage(c, cmd); got != helpText {
		t.Errorf("GetHelpPage(%q) = %q; want %q", cmd, got, helpText)
	}
}

func TestCacheExpiration(t *testing.T) {
	// Create a cache with a very short expiration time to test expiry behavior.
	c := cache.New(100*time.Millisecond, 50*time.Millisecond)
	cmd := "expiringCommand"
	helpText := "This help text should expire soon."

	// Cache the help text with short expiration
	c.Set(cmd, helpText, 100*time.Millisecond)

	// Immediately after caching, the text should be retrievable.
	if got := GetHelpPage(c, cmd); got != helpText {
		t.Errorf("GetHelpPage(%q) = %q; want %q", cmd, got, helpText)
	}

	// Wait longer than the expiration duration.
	time.Sleep(150 * time.Millisecond)

	// Now, the help text should have expired and not be retrievable.
	if got := GetHelpPage(c, cmd); got != "" {
		t.Errorf("After expiration, GetHelpPage(%q) = %q; want empty string", cmd, got)
	}
}
