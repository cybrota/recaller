// avl_tree.go

package main

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

func (tree *AVLTree) Insert(key string, value interface{}) {
	tree.Root = tree._insert(tree.Root, key, value)
}

func (tree *AVLTree) _insert(node *AVLNode, key string, value interface{}) *AVLNode {
	if node == nil {
		return &AVLNode{Key: key, Value: value, Height: 1}
	}

	if key < node.Key {
		node.Left = tree._insert(node.Left, key, value)
	} else if key > node.Key {
		node.Right = tree._insert(node.Right, key, value)
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
	tree.Root = tree._deleteRecursive(tree.Root, key)
}

func (tree *AVLTree) _deleteRecursive(node *AVLNode, key string) *AVLNode {
	if node == nil {
		return nil // Key not found
	}

	if key < node.Key {
		node.Left = tree._deleteRecursive(node.Left, key)
	} else if key > node.Key {
		node.Right = tree._deleteRecursive(node.Right, key)
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
		node.Right = tree._deleteRecursive(node.Right, pivot.Key)
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
