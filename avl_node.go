package main

type AVLNode struct {
	Key    string      // Command (e.g., "echo Hello, World!")
	Value  interface{} // Associated data (e.g., timestamp)
	Height int
	Left   *AVLNode
	Right  *AVLNode
}
