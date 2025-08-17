// cache.go

/**
 * Copyright 2025 (C) Naren Yellavula - All Rights Reserved
 *
 * This source code is protected under international copyright law.  All rights
 * reserved and protected by the copyright holders.
 * This file is confidential and only available to authorized individuals with the
 * permission of the copyright holders.  If you encounter this file and do not have
 * permission, please contact the copyright holders and delete this file.
 */

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
