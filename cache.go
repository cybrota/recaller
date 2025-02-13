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
)

func CacheHelpPage(c *cache.Cache, cmd string, helpTxt string) {
	c.Add(cmd, helpTxt, cache.DefaultExpiration)
}

func GetHelpPage(c *cache.Cache, cmd string) string {
	val, ok := c.Get(cmd)
	if !ok {
		return ""
	}
	return val.(string)
}
