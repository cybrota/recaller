// avl_tree_test.go

package main

import (
	"testing"
)

type AVLTestCase struct {
	Name          string
	InitialKeys   []string
	KeysToInsert  []string
	KeysToDelete  []string
	ExpectedOrder []string // In-order traversal expectation after operations
}

func TestAVLTreeOperations(t *testing.T) {
	testCases := []AVLTestCase{
		{
			Name:          "Simple Insertion",
			InitialKeys:   nil,
			KeysToInsert:  []string{"apple", "banana", "cherry"},
			ExpectedOrder: []string{"apple", "banana", "cherry"},
		},
		{
			Name:          "Insertion with Balancing (Left-Heavy)",
			InitialKeys:   []string{"apple"},
			KeysToInsert:  []string{"banana", "cherry"},
			ExpectedOrder: []string{"apple", "banana", "cherry"},
		},
		{
			Name:          "Deletion with Balancing (Right-Heavy)",
			InitialKeys:   []string{"cherry", "banana", "apple"},
			KeysToDelete:  []string{"cherry"},
			ExpectedOrder: []string{"apple", "banana"},
		},
		{
			Name:          "Mixed Operations",
			InitialKeys:   []string{"dog", "cat"},
			KeysToInsert:  []string{"elephant", "bird"},
			KeysToDelete:  []string{"cat"},
			ExpectedOrder: []string{"bird", "dog", "elephant"},
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tree := NewAVLTree()
			// Insert initial keys
			for _, key := range tc.InitialKeys {
				tree.Insert(key, CommandMetadata{})
			}
			// Perform insert operations
			for _, key := range tc.KeysToInsert {
				tree.Insert(key, CommandMetadata{})
			}
			// Perform delete operations
			for _, key := range tc.KeysToDelete {
				tree.Delete(key)
			}
			// Verify the in-order traversal matches expectations
			if !verifyInOrderTraversal(t, tree.Root, tc.ExpectedOrder) {
				t.Errorf("In-order traversal mismatch for test case '%s'", tc.Name)
			}
		})
	}
}

func verifyInOrderTraversal(t *testing.T, node *AVLNode, expected []string) bool {
	var actual []string
	inOrderTraversal(node, &actual)
	if len(actual) != len(expected) {
		t.Logf("Length mismatch. Expected %d elements, got %d", len(expected), len(actual))
		return false
	}
	for i := range expected {
		if actual[i] != expected[i] {
			t.Logf("Mismatch at index %d. Expected '%s', got '%s'", i, expected[i], actual[i])
			return false
		}
	}
	return true
}

func inOrderTraversal(node *AVLNode, result *[]string) {
	if node == nil {
		return
	}
	inOrderTraversal(node.Left, result)
	*result = append(*result, node.Key)
	inOrderTraversal(node.Right, result)
}
