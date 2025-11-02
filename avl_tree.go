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
	"sort"
	"strings"
	"time"
)

type CommandMetadata struct {
	Command   string
	Timestamp *time.Time // Unix timestamp for recency (updated on each use)
	Frequency int        // Incremented on each command execution
}

type RankedCommand struct {
	Command  string
	Score    float64
	Metadata CommandMetadata
}

type AVLNode struct {
	Key    string          // Command (e.g., "echo Hello, World!")
	Value  CommandMetadata // Associated data (e.g., timestamp)
	Height int
	Left   *AVLNode
	Right  *AVLNode
}

type AVLTreeIFace interface {
	Insert(key string, value interface{})
	Delete(key string)
	Search(key string) (interface{}, bool)
	SearchPrefix(prefix string) []*AVLNode
	SearchFuzzy(query string) []*AVLNode
	SearchPrefixMostRecent(prefix string) []*AVLNode
}

type AVLTree struct {
	Root *AVLNode
}

func NewAVLTree() *AVLTree {
	return &AVLTree{Root: nil}
}

func (tree *AVLTree) getHeight(node *AVLNode) int {
	if node == nil {
		return 0
	}
	return node.Height
}

func (tree *AVLTree) updateHeight(node *AVLNode) {
	node.Height = max(tree.getHeight(node.Left), tree.getHeight(node.Right)) + 1
}

func (tree *AVLTree) getBalanceFactor(node *AVLNode) int {
	if node == nil {
		return 0
	}
	return tree.getHeight(node.Left) - tree.getHeight(node.Right)
}

func (tree *AVLTree) rotateLeft(node *AVLNode) *AVLNode {
	// Check if input node is valid
	if node == nil || node.Right == nil {
		return node // Nothing to rotate or invalid input
	}

	// Identify the pivot node (new root)
	pivot := node.Right

	// Perform the rotation
	node.Right = pivot.Left
	pivot.Left = node

	// Update heights
	tree.updateHeight(node)
	tree.updateHeight(pivot)

	return pivot // Return the new root node
}

func (tree *AVLTree) rotateRight(node *AVLNode) *AVLNode {
	// Check if input node is valid
	if node == nil || node.Left == nil {
		return node // Nothing to rotate or invalid input
	}

	// Identify the pivot node (new root)
	pivot := node.Left

	// Perform the rotation
	node.Left = pivot.Right
	pivot.Right = node

	// Update heights
	tree.updateHeight(node)
	tree.updateHeight(pivot)

	return pivot // Return the new root node
}

func (tree *AVLTree) Insert(key string, value CommandMetadata) {
	tree.Root = tree.insertRecursive(tree.Root, key, value)
}

func (tree *AVLTree) insertRecursive(node *AVLNode, key string, value CommandMetadata) *AVLNode {
	if node == nil {
		return &AVLNode{Key: key, Value: value, Height: 1}
	}

	if key < node.Key {
		node.Left = tree.insertRecursive(node.Left, key, value)
	} else if key > node.Key {
		node.Right = tree.insertRecursive(node.Right, key, value)
	} else {
		// Handle duplicate keys (e.g., update value)
	}

	tree.updateHeight(node)

	balanceFactor := tree.getBalanceFactor(node)
	if balanceFactor > 1 {
		if key < node.Left.Key {
			return tree.rotateRight(node)
		} else {
			// Left-Right case
			node.Left = tree.rotateLeft(node.Left)
			return tree.rotateRight(node)
		}
	} else if balanceFactor < -1 {
		if key > node.Right.Key {
			return tree.rotateLeft(node)
		} else {
			// Right-Left case
			node.Right = tree.rotateRight(node.Right)
			return tree.rotateLeft(node)
		}
	}

	return node
}

func (tree *AVLTree) Delete(key string) {
	tree.Root = tree.deleteRecursive(tree.Root, key)
}

func (tree *AVLTree) deleteRecursive(node *AVLNode, key string) *AVLNode {
	if node == nil {
		return nil // Key not found
	}

	if key < node.Key {
		node.Left = tree.deleteRecursive(node.Left, key)
	} else if key > node.Key {
		node.Right = tree.deleteRecursive(node.Right, key)
	} else { // Found the node to delete
		// Case 1: No children
		if node.Left == nil && node.Right == nil {
			return nil
		}
		// Case 2: One child (right)
		if node.Left == nil {
			return node.Right
		}
		// Case 3: One child (left)
		if node.Right == nil {
			return node.Left
		}
		// Case 4: Two children
		pivot := tree.findMin(node.Right) // Find the minimum in the right subtree
		node.Key = pivot.Key
		node.Value = pivot.Value
		node.Right = tree.deleteRecursive(node.Right, pivot.Key)
	}

	// Update height and balance factor after deletion
	tree.updateHeight(node)
	return tree.rebalance(node)
}

func (tree *AVLTree) findMin(node *AVLNode) *AVLNode {
	for node.Left != nil {
		node = node.Left
	}
	return node
}

func (tree *AVLTree) rebalance(node *AVLNode) *AVLNode {
	balanceFactor := tree.getBalanceFactor(node)

	// Left-heavy
	if balanceFactor > 1 {
		if tree.getBalanceFactor(node.Left) >= 0 {
			return tree.rotateRight(node)
		} else {
			node.Left = tree.rotateLeft(node.Left)
			return tree.rotateRight(node)
		}
	}

	// Right-heavy
	if balanceFactor < -1 {
		if tree.getBalanceFactor(node.Right) <= 0 {
			return tree.rotateLeft(node)
		} else {
			node.Right = tree.rotateRight(node.Right)
			return tree.rotateLeft(node)
		}
	}

	return node
}

// Search looks for the node with the given key in the AVL tree.
// It returns the value if found, and a boolean indicating whether the key was found.
func (tree *AVLTree) Search(key string) (interface{}, bool) {
	return searchNode(tree.Root, key)
}

// searchNode is a helper function that traverses the AVL tree recursively.
func searchNode(node *AVLNode, key string) (interface{}, bool) {
	if node == nil {
		return nil, false
	}

	if key < node.Key {
		return searchNode(node.Left, key)
	} else if key > node.Key {
		return searchNode(node.Right, key)
	} else {
		// key == node.Key
		return node.Value, true
	}
}

// rangeSearch traverses the subtree rooted at 'node' and appends to 'results'
// every node whose Key satisfies low <= Key < high, in ascending (lexicographical) order.
func rangeSearch(node *AVLNode, low, high string, results *[]*AVLNode) {
	if node == nil {
		return
	}

	// Use string comparison optimization - only traverse left if needed
	if node.Key >= low {
		rangeSearch(node.Left, low, high, results)
	}

	// If node.Key is actually in [low, high), collect it
	if len(node.Key) >= len(low) && strings.HasPrefix(node.Key, low) {
		*results = append(*results, node)
	}

	// Use string comparison optimization - only traverse right if needed
	if node.Key < high {
		rangeSearch(node.Right, low, high, results)
	}
}

func (tree *AVLTree) SearchPrefix(prefix string) []*AVLNode {
	var results []*AVLNode
	// Construct high bound as prefix + "\uffff"
	high := prefix + "\uffff"

	rangeSearch(tree.Root, prefix, high, &results)
	return results
}

func (tree *AVLTree) SearchPrefixMostRecent(prefix string) []*AVLNode {
	// 1. Gather prefix matches (keys in [prefix, prefix+"\uffff"))
	matches := tree.SearchPrefix(prefix)

	sort.Slice(matches, func(i, j int) bool {
		// Type assert both sides to *time.Time
		t1 := matches[i].Value.Timestamp
		t2 := matches[j].Value.Timestamp

		if t1 == nil && t2 == nil {
			return false
		}
		if t1 == nil {
			// nil is considered older
			return false
		}
		if t2 == nil {
			// non-nil is considered newer
			return true
		}

		// Now both t1, t2 are non-nil *time.Time
		// Return true if t1 is after t2 => t1 is more recent
		return t1.After(*t2)
	})

	return matches
}

func calculateScore(metadata CommandMetadata) float64 {
	frequencyScore := float64(metadata.Frequency)

	var recencyScore float64
	if metadata.Timestamp != nil && !metadata.Timestamp.IsZero() {
		timeDelta := time.Since(*metadata.Timestamp).Hours()
		if timeDelta < 0 {
			timeDelta = 0
		}
		recencyScore = 1 / (timeDelta + 1) // Add 1 to avoid division by zero
	}

	return (0.6 * frequencyScore) + (0.4 * recencyScore)
}

// fuzzySearch performs in-order traversal and finds commands containing the query as substring
func fuzzySearch(node *AVLNode, query string, results *[]*AVLNode) {
	if node == nil {
		return
	}

	// Traverse left subtree
	fuzzySearch(node.Left, query, results)

	// Check if current node contains the query as substring (case-insensitive)
	if strings.Contains(strings.ToLower(node.Key), strings.ToLower(query)) {
		*results = append(*results, node)
	}

	// Traverse right subtree
	fuzzySearch(node.Right, query, results)
}

func (tree *AVLTree) SearchFuzzy(query string) []*AVLNode {
	var results []*AVLNode
	fuzzySearch(tree.Root, query, &results)
	return results
}

func SearchWithRanking(tree *AVLTree, query string, enableFuzzing bool) []RankedCommand {
	var nodes []*AVLNode

	if enableFuzzing {
		nodes = tree.SearchFuzzy(query)
	} else {
		nodes = tree.SearchPrefix(query)
	}

	// Pre-allocate slice with estimated capacity to reduce allocations
	rankedCommands := make([]RankedCommand, 0, len(nodes))

	// Traverse the tree to find matching commands
	for _, node := range nodes {
		command := node.Key
		metadata := node.Value

		rankedCommand := RankedCommand{
			Command:  command,
			Score:    calculateScore(metadata),
			Metadata: metadata, // Reuse existing metadata to avoid copying
		}

		rankedCommands = append(rankedCommands, rankedCommand)
	}

	// Sort the commands based on their scores (Descending order for highest score first)
	sort.SliceStable(rankedCommands, func(i, j int) bool {
		return rankedCommands[i].Score > rankedCommands[j].Score
	})

	return rankedCommands
}
