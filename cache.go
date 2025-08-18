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
	"github.com/patrickmn/go-cache"
	"time"
)

const (
	// Cache help pages for 30 minutes instead of default (which can be much longer)
	helpCacheExpiration = 30 * time.Minute
	// Clean up expired entries every 5 minutes
	helpCacheCleanup = 5 * time.Minute
)

// NewOptimizedHelpCache creates a cache optimized for help text storage
func NewOptimizedHelpCache() *cache.Cache {
	return cache.New(helpCacheExpiration, helpCacheCleanup)
}

func CacheHelpPage(c *cache.Cache, cmd string, helpTxt string) {
	// Use Set instead of Add to allow overwriting (more efficient for repeated commands)
	c.Set(cmd, helpTxt, helpCacheExpiration)
}

func GetHelpPage(c *cache.Cache, cmd string) string {
	val, ok := c.Get(cmd)
	if !ok {
		return ""
	}
	return val.(string)
}
