// quotes.go

/**
 * Copyright (C) Naren Yellavula - All Rights Reserved
 *
 * This source code is protected under international copyright law.  All rights
 * reserved and protected by the copyright holders.
 * This file is confidential and only available to authorized individuals with the
 * permission of the copyright holders.  If you encounter this file and do not have
 * permission, please contact the copyright holders and delete this file.
 * The quotations are taken from the public domain and attributed to respective creators.
 */

package main

import (
	"math/rand"
)

var quotes = []string{
	"We are not Rational beings",
	"Never say never",
	"You can't change people, only youself",
	"Act like you're evolved: breathe, don't hiss",
	"Trust intuition, but verify",
	"Objectives move you to your goal",
	"Everything is a trade-off",
	"Time can't be created or destroyed, only allocated",
	"When stuck, talk to the duck",
	"Read deliberately",
	"Chance favors the prepared mind",
	"Learn from similarities, unlearn from differences",
	"90% of the functionality delivered now is better than 100% delivered never.",
	"Don't document bad code - rewrite it.",
	"Write boring code to save yourself from debugging later",
	"If you had done something twice, \nyou are likely to do it again",
	"When in doubt, use brute force",
	"We already have persistent objects, they're called files",
	"I am a very bottom-up thinker starting from the top",
	"Narrowness of experience leads to narrowness of imagination",
	"Caches aren't architecture, they're just optimization",
	"Always deliver more than expected",
	"Find the leverage in the world so you can be truly lazy",
	"You can be serious without a suit",
	"If you optimize everything, you will always be unhappy",
	"Some problems are better evaded than solved",
	"Premature optimization is the root of all evil",
	"Avoiding complexity reduces bugs",
	"People will realize that software is not a product; \nyou use it to build a product",
	"Any program is only as good as it is useful",
	"If you have the right attitude, \ninteresting problems will find you",
	"Prototype, then polish. Get it working before you optimize it",
	"Do have fun sometimes",
	"Release early. Release often. And listen to your customers",
}

// pickRandomString returns a random string from the provided slice.
// If the slice is empty, it returns an empty string.
func pickRandomString(list []string) string {
	// If the list is empty, return empty string.
	if len(list) == 0 {
		return ""
	}
	// Generate a random index and return the element at that index.
	randomIndex := rand.Intn(len(list))
	return list[randomIndex]
}

func GetRandomQuote() string {
	return pickRandomString(quotes)
}
