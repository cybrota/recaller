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
