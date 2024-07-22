package trie

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/mpetrun5/merkle-patricia-trie/nibble"
	"github.com/mpetrun5/merkle-patricia-trie/node"
	"github.com/mpetrun5/merkle-patricia-trie/proof"
)

type Trie struct {
	root node.Node
}

func NewTrie() *Trie {
	return &Trie{}
}

func (t *Trie) Hash() []byte {
	if node.IsEmptyNode(t.root) {
		return node.EmptyNodeHash
	}
	return t.root.Hash()
}

func (t *Trie) Get(key []byte) ([]byte, bool) {
	root := t.root
	nibbles := nibble.FromBytes(key)
	for {
		if node.IsEmptyNode(root) {
			return nil, false
		}

		if leaf, ok := root.(*node.LeafNode); ok {
			matched := nibble.PrefixMatchedLen(leaf.Path, nibbles)
			if matched != len(leaf.Path) || matched != len(nibbles) {
				return nil, false
			}
			return leaf.Value, true
		}

		if branch, ok := root.(*node.BranchNode); ok {
			if len(nibbles) == 0 {
				return branch.Value, branch.HasValue()
			}

			b, remaining := nibbles[0], nibbles[1:]
			nibbles = remaining
			root = branch.Branches[b]
			continue
		}

		if ext, ok := root.(*node.ExtensionNode); ok {
			matched := nibble.PrefixMatchedLen(ext.Path, nibbles)
			// E 01020304
			//   010203
			if matched < len(ext.Path) {
				return nil, false
			}

			nibbles = nibbles[matched:]
			root = ext.Next
			continue
		}

		panic("not found")
	}
}

// Put adds a key value pair to the trie
// In general, the rule is:
// - When stopped at an EmptyNode, replace it with a new LeafNode with the remaining path.
// - When stopped at a LeafNode, convert it to an ExtensionNode and add a new branch and a new LeafNode.
// - When stopped at an ExtensionNode, convert it to another ExtensionNode with shorter path and create a new BranchNode points to the ExtensionNode.
func (t *Trie) Put(key []byte, value []byte) {
	// need to use pointer, so that I can update root in place without
	// keeping trace of the parent node
	root := &t.root
	nibbles := nibble.FromBytes(key)
	for {
		if node.IsEmptyNode(*root) {
			leaf := node.NewLeafNodeFromNibbles(nibbles, value)
			*root = leaf
			return
		}

		if leaf, ok := (*root).(*node.LeafNode); ok {
			matched := nibble.PrefixMatchedLen(leaf.Path, nibbles)

			// if all matched, update value even if the value are equal
			if matched == len(nibbles) && matched == len(leaf.Path) {
				newLeaf := node.NewLeafNodeFromNibbles(leaf.Path, value)
				*root = newLeaf
				return
			}

			branch := node.NewBranchNode()
			// if matched some nibbles, check if matches either all remaining nibbles
			// or all leaf nibbles
			if matched == len(leaf.Path) {
				branch.SetValue(leaf.Value)
			}

			if matched == len(nibbles) {
				branch.SetValue(value)
			}

			// if there is matched nibbles, an extension node will be created
			if matched > 0 {
				// create an extension node for the shared nibbles
				ext := node.NewExtensionNode(leaf.Path[:matched], branch)
				*root = ext
			} else {
				// when there no matched nibble, there is no need to keep the extension node
				*root = branch
			}

			if matched < len(leaf.Path) {
				// have dismatched
				// L 01020304 hello
				// + 010203   world

				// 01020304, 0, 4
				branchNibble, leafNibbles := leaf.Path[matched], leaf.Path[matched+1:]
				newLeaf := node.NewLeafNodeFromNibbles(leafNibbles, leaf.Value) // not :matched+1
				branch.SetBranch(branchNibble, newLeaf)
			}

			if matched < len(nibbles) {
				// L 01020304 hello
				// + 010203040 world

				// L 01020304 hello
				// + 010203040506 world
				branchNibble, leafNibbles := nibbles[matched], nibbles[matched+1:]
				newLeaf := node.NewLeafNodeFromNibbles(leafNibbles, value)
				branch.SetBranch(branchNibble, newLeaf)
			}

			return
		}

		if branch, ok := (*root).(*node.BranchNode); ok {
			if len(nibbles) == 0 {
				branch.SetValue(value)
				return
			}

			b, remaining := nibbles[0], nibbles[1:]
			nibbles = remaining
			root = &branch.Branches[b]
			continue
		}

		// E 01020304
		// B 0 hello
		// L 506 world
		// + 010203 good
		if ext, ok := (*root).(*node.ExtensionNode); ok {
			matched := nibble.PrefixMatchedLen(ext.Path, nibbles)
			if matched < len(ext.Path) {
				// E 01020304
				// + 010203 good
				extNibbles, branchNibble, extRemainingnibbles := ext.Path[:matched], ext.Path[matched], ext.Path[matched+1:]
				branch := node.NewBranchNode()
				if len(extRemainingnibbles) == 0 {
					// E 0102030
					// + 010203 good
					branch.SetBranch(branchNibble, ext.Next)
				} else {
					// E 01020304
					// + 010203 good
					newExt := node.NewExtensionNode(extRemainingnibbles, ext.Next)
					branch.SetBranch(branchNibble, newExt)
				}

				if matched < len(nibbles) {
					nodeBranchNibble, nodeLeafNibbles := nibbles[matched], nibbles[matched+1:]
					remainingLeaf := node.NewLeafNodeFromNibbles(nodeLeafNibbles, value)
					branch.SetBranch(nodeBranchNibble, remainingLeaf)
				} else if matched == len(nibbles) {
					branch.SetValue(value)
				} else {
					panic(fmt.Sprintf("too many matched (%v > %v)", matched, len(nibbles)))
				}

				// if there is no shared extension nibbles any more, then we don't need the extension node
				// any more
				// E 01020304
				// + 1234 good
				if len(extNibbles) == 0 {
					*root = branch
				} else {
					// otherwise create a new extension node
					*root = node.NewExtensionNode(extNibbles, branch)
				}
				return
			}

			nibbles = nibbles[matched:]
			root = &ext.Next
			continue
		}

		panic("unknown type")
	}

}

// Prove returns the merkle proof for the given key, which is
func (t *Trie) Prove(key []byte) (proof.Proof, bool) {
	proof := proof.NewProofDB()
	root := t.root
	nibbles := nibble.FromBytes(key)

	for {
		proof.Put(node.Hash(root), node.Serialize(root))

		if node.IsEmptyNode(root) {
			return nil, false
		}

		if leaf, ok := root.(*node.LeafNode); ok {
			matched := nibble.PrefixMatchedLen(leaf.Path, nibbles)
			if matched != len(leaf.Path) || matched != len(nibbles) {
				return nil, false
			}

			return proof, true
		}

		if branch, ok := root.(*node.BranchNode); ok {
			if len(nibbles) == 0 {
				return proof, branch.HasValue()
			}

			b, remaining := nibbles[0], nibbles[1:]
			nibbles = remaining
			root = branch.Branches[b]
			continue
		}

		if ext, ok := root.(*node.ExtensionNode); ok {
			matched := nibble.PrefixMatchedLen(ext.Path, nibbles)
			// E 01020304
			//   010203
			if matched < len(ext.Path) {
				return nil, false
			}

			nibbles = nibbles[matched:]
			root = ext.Next
			continue
		}

		panic("not found")
	}
}

// VerifyProof verify the proof for the given key under the given root hash using go-ethereum's VerifyProof implementation.
// It returns the value for the key if the proof is valid, otherwise error will be returned
func VerifyProof(rootHash []byte, key []byte, proof proof.Proof) (value []byte, err error) {
	return trie.VerifyProof(common.BytesToHash(rootHash), key, proof)
}
