// history_tree.go

package main

import (
	"errors"
	"sort"
	"time"
)

// Node represents an individual element within the BBST.
type Node struct {
    Command    string
    Timestamp  time.Time
    left, right *Node
    height      int
}

// HistoryTree is a basic Balanced Binary Search Tree for shell history.
type HistoryTree struct {
    root *Node
}

// NewHistoryTree returns a new instance of the HistoryTree.
func NewHistoryTree() *HistoryTree {
    return &HistoryTree{root: nil}
}

// Insert adds a new command to the tree, maintaining balance.
func (ht *HistoryTree) Insert(command string) error {
    newNode := &Node{
        Command:   command,
        Timestamp: time.Now(),
        height:    1,
    }
    if ht.root == nil {
        ht.root = newNode
        return nil
    }
    // Basic insertion, to be extended with balancing logic (e.g., AVL or Red-Black tree properties)
    ht.insertNode(ht.root, newNode)
    return nil
}

func (ht *HistoryTree) insertNode(current *Node, newNode *Node) {
    if newNode.Command < current.Command {
        if current.left == nil {
            current.left = newNode
        } else {
            // Recurse and balance later
            ht.insertNode(current.left, newNode)
        }
    } else if newNode.Command > current.Command {
        if current.right == nil {
            current.right = newNode
        } else {
            // Recurse and balance later
            ht.insertNode(current.right, newNode)
        }
    } else {
        // Handling duplicate commands (consider appending timestamp or not allowing duplicates)
        return
    }
}

func (ht *HistoryTree) GetRecentCommands(n int) ([]string, error) {
    if ht.root == nil {
        return []string{}, errors.New("history is empty")
    }
    var allCommands []*Node // Store nodes in the order they're visited
    ht.inOrderTraversalWithTime(ht.root, &allCommands)

    // Sort the commands by timestamp in descending order (newest first)
    sort.Slice(allCommands, func(i, j int) bool {
        return allCommands[i].Timestamp.After(allCommands[j].Timestamp)
    })

    recentCmds := make([]string, 0, n)
    for _, node := range allCommands[:n] { // Get the N most recent
        recentCmds = append(recentCmds, node.Command)
    }
    return recentCmds, nil
}

func (ht *HistoryTree) inOrderTraversalWithTime(node *Node, nodes *[]*Node) {
    if node != nil {
        ht.inOrderTraversalWithTime(node.left, nodes)
        *nodes = append(*nodes, node) // Collect all nodes for later sorting
        ht.inOrderTraversalWithTime(node.right, nodes)
    }
}
