package merkletree

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
)

// Node represents a node in the Merkle tree
type Node struct {
	Hash  string
	Left  *Node
	Right *Node
}

// MerkleTree implements Merkle tree hashing for document verification
type MerkleTree struct {
	Root *Node
}

// NewMerkleTree creates a new Merkle tree from a list of data strings
func NewMerkleTree(data []string) (*MerkleTree, error) {
	if len(data) == 0 {
		return nil, errors.New("cannot create Merkle tree with no data")
	}

	// Create leaf nodes
	var nodes []*Node
	for _, item := range data {
		hash := sha256Hash(item)
		nodes = append(nodes, &Node{Hash: hash})
	}

	// If odd number of nodes, duplicate the last one
	if len(nodes)%2 != 0 && len(nodes) > 1 {
		lastNode := nodes[len(nodes)-1]
		nodes = append(nodes, &Node{Hash: lastNode.Hash})
	}

	// Build the tree from bottom up
	root := buildTree(nodes)
	return &MerkleTree{Root: root}, nil
}

// buildTree builds the Merkle tree from a list of leaf nodes
func buildTree(nodes []*Node) *Node {
	if len(nodes) == 1 {
		return nodes[0]
	}

	var newLevel []*Node

	// Process pairs of nodes
	for i := 0; i < len(nodes); i += 2 {
		if i+1 < len(nodes) {
			// Create parent node with two children
			parentHash := sha256Hash(nodes[i].Hash + nodes[i+1].Hash)
			parent := &Node{
				Hash:  parentHash,
				Left:  nodes[i],
				Right: nodes[i+1],
			}
			newLevel = append(newLevel, parent)
		} else {
			// If odd number, promote the single node
			newLevel = append(newLevel, nodes[i])
		}
	}

	// Recursively build the next level
	return buildTree(newLevel)
}

// VerifyDocument verifies if a document is part of the Merkle tree
func (m *MerkleTree) VerifyDocument(document string, proof []string, index int) bool {
	if m.Root == nil {
		return false
	}

	// Calculate the hash of the document
	currentHash := sha256Hash(document)

	// Navigate through the proof
	for _, proofItem := range proof {
		// Determine if this proof hash is on the left or right
		if index%2 == 0 {
			// Current hash is on the left, proof hash is on the right
			currentHash = sha256Hash(currentHash + proofItem)
		} else {
			// Current hash is on the right, proof hash is on the left
			currentHash = sha256Hash(proofItem + currentHash)
		}
		// Move up to the parent index
		index /= 2
	}

	// Check if the calculated root hash matches the stored root hash
	return currentHash == m.Root.Hash
}

// GenerateProof generates a proof for a document at a specific index
func (m *MerkleTree) GenerateProof(data []string, index int) ([]string, error) {
	if index < 0 || index >= len(data) {
		return nil, errors.New("index out of range")
	}

	// Calculate total number of leaf nodes (rounded to next power of 2)
	leafCount := int(math.Pow(2, math.Ceil(math.Log2(float64(len(data))))))
	
	// Create a map to store nodes by their position
	nodeMap := make(map[int]*Node)
	
	// Create leaf nodes and add to map
	for i, item := range data {
		hash := sha256Hash(item)
		nodeMap[leafCount-1+i] = &Node{Hash: hash}
	}
	
	// Fill in any missing leaf nodes with duplicates of the last node
	for i := len(data); i < leafCount; i++ {
		nodeMap[leafCount-1+i] = nodeMap[leafCount-1+len(data)-1]
	}
	
	// Build non-leaf nodes
	for i := leafCount - 2; i >= 0; i-- {
		leftChild := nodeMap[2*i+1]
		rightChild := nodeMap[2*i+2]
		if leftChild != nil && rightChild != nil {
			nodeMap[i] = &Node{
				Hash:  sha256Hash(leftChild.Hash + rightChild.Hash),
				Left:  leftChild,
				Right: rightChild,
			}
		}
	}
	
	// Generate proof
	proof := []string{}
	idx := leafCount - 1 + index
	
	for idx > 0 {
		var siblingIdx int
		if idx%2 == 0 {
			siblingIdx = idx - 1 // Left sibling
		} else {
			siblingIdx = idx + 1 // Right sibling
		}
		
		if sibling, ok := nodeMap[siblingIdx]; ok && sibling != nil {
			proof = append(proof, sibling.Hash)
		}
		
		// Move up to parent
		idx = (idx - 1) / 2
	}
	
	return proof, nil
}

// GetRootHash returns the Merkle root hash
func (m *MerkleTree) GetRootHash() string {
	if m.Root == nil {
		return ""
	}
	return m.Root.Hash
}

// sha256Hash computes the SHA-256 hash of a string
func sha256Hash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
