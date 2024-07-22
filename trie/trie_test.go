package trie

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/trie"
	"github.com/mpetrun5/merkle-patricia-trie/nibble"
	"github.com/mpetrun5/merkle-patricia-trie/node"
	"github.com/mpetrun5/merkle-patricia-trie/proof"
	"github.com/stretchr/testify/require"
)

func hexEqual(t *testing.T, hex string, bytes []byte) {
	require.Equal(t, hex, fmt.Sprintf("%x", bytes))
}

// check basic key-value mapping
func TestGetPut(t *testing.T) {
	t.Run("should get nothing if key does not exist", func(t *testing.T) {
		trie := NewTrie()
		_, found := trie.Get([]byte("notexist"))
		require.Equal(t, false, found)
	})

	t.Run("should get value if key exist", func(t *testing.T) {
		trie := NewTrie()
		trie.Put([]byte{1, 2, 3, 4}, []byte("hello"))
		val, found := trie.Get([]byte{1, 2, 3, 4})
		require.Equal(t, true, found)
		require.Equal(t, val, []byte("hello"))
	})

	t.Run("should get updated value", func(t *testing.T) {
		trie := NewTrie()
		trie.Put([]byte{1, 2, 3, 4}, []byte("hello"))
		trie.Put([]byte{1, 2, 3, 4}, []byte("world"))
		val, found := trie.Get([]byte{1, 2, 3, 4})
		require.Equal(t, true, found)
		require.Equal(t, val, []byte("world"))
	})
}

// verify data integrity
func TestDataIntegrity(t *testing.T) {
	t.Run("should get a different hash if a new key-value pair was added or updated", func(t *testing.T) {
		trie := NewTrie()
		hash0 := trie.Hash()

		trie.Put([]byte{1, 2, 3, 4}, []byte("hello"))
		hash1 := trie.Hash()

		trie.Put([]byte{1, 2}, []byte("world"))
		hash2 := trie.Hash()

		trie.Put([]byte{1, 2}, []byte("trie"))
		hash3 := trie.Hash()

		require.NotEqual(t, hash0, hash1)
		require.NotEqual(t, hash1, hash2)
		require.NotEqual(t, hash2, hash3)
	})

	t.Run("should get the same hash if two tries have the identicial key-value pairs", func(t *testing.T) {
		trie1 := NewTrie()
		trie1.Put([]byte{1, 2, 3, 4}, []byte("hello"))
		trie1.Put([]byte{1, 2}, []byte("world"))

		trie2 := NewTrie()
		trie2.Put([]byte{1, 2, 3, 4}, []byte("hello"))
		trie2.Put([]byte{1, 2}, []byte("world"))

		require.Equal(t, trie1.Hash(), trie2.Hash())
	})
}

func TestPut2Pairs(t *testing.T) {
	trie := NewTrie()
	trie.Put([]byte{1, 2, 3, 4}, []byte("verb"))
	trie.Put([]byte{1, 2, 3, 4, 5, 6}, []byte("coin"))

	verb, ok := trie.Get([]byte{1, 2, 3, 4})
	require.True(t, ok)
	require.Equal(t, []byte("verb"), verb)

	coin, ok := trie.Get([]byte{1, 2, 3, 4, 5, 6})
	require.True(t, ok)
	require.Equal(t, []byte("coin"), coin)

	fmt.Printf("%T\n", trie.root)
	ext, ok := trie.root.(*node.ExtensionNode)
	require.True(t, ok)
	branch, ok := ext.Next.(*node.BranchNode)
	require.True(t, ok)
	leaf, ok := branch.Branches[0].(*node.LeafNode)
	require.True(t, ok)

	hexEqual(t, "c37ec985b7a88c2c62beb268750efe657c36a585beb435eb9f43b839846682ce", leaf.Hash())
	hexEqual(t, "ddc882350684636f696e8080808080808080808080808080808476657262", branch.Serialize())
	hexEqual(t, "d757709f08f7a81da64a969200e59ff7e6cd6b06674c3f668ce151e84298aa79", branch.Hash())
	hexEqual(t, "64d67c5318a714d08de6958c0e63a05522642f3f1087c6fd68a97837f203d359", ext.Hash())
}

func TestPut(t *testing.T) {
	trie := NewTrie()
	require.Equal(t, node.EmptyNodeHash, trie.Hash())
	trie.Put([]byte{1, 2, 3, 4}, []byte("hello"))
	ns := node.NewLeafNodeFromBytes([]byte{1, 2, 3, 4}, []byte("hello"))
	require.Equal(t, ns.Hash(), trie.Hash())
}

func TestPutLeafShorter(t *testing.T) {
	trie := NewTrie()
	trie.Put([]byte{1, 2, 3, 4}, []byte("hello"))
	trie.Put([]byte{1, 2, 3}, []byte("world"))

	leaf := node.NewLeafNodeFromNibbles([]nibble.Nibble{4}, []byte("hello"))

	branch := node.NewBranchNode()
	branch.SetBranch(nibble.Nibble(0), leaf)
	branch.SetValue([]byte("world"))

	ext := node.NewExtensionNode([]nibble.Nibble{0, 1, 0, 2, 0, 3}, branch)

	require.Equal(t, ext.Hash(), trie.Hash())
}

func TestPutLeafAllMatched(t *testing.T) {
	trie := NewTrie()
	trie.Put([]byte{1, 2, 3, 4}, []byte("hello"))
	trie.Put([]byte{1, 2, 3, 4}, []byte("world"))

	ns := node.NewLeafNodeFromBytes([]byte{1, 2, 3, 4}, []byte("world"))
	require.Equal(t, ns.Hash(), trie.Hash())
}

func TestPutLeafMore(t *testing.T) {
	trie := NewTrie()
	trie.Put([]byte{1, 2, 3, 4}, []byte("hello"))
	trie.Put([]byte{1, 2, 3, 4, 5, 6}, []byte("world"))

	leaf := node.NewLeafNodeFromNibbles([]nibble.Nibble{5, 0, 6}, []byte("world"))

	branch := node.NewBranchNode()
	branch.SetValue([]byte("hello"))
	branch.SetBranch(nibble.Nibble(0), leaf)

	ext := node.NewExtensionNode([]nibble.Nibble{0, 1, 0, 2, 0, 3, 0, 4}, branch)

	require.Equal(t, ext.Hash(), trie.Hash())
}

func TestPutOrder(t *testing.T) {
	trie1, trie2 := NewTrie(), NewTrie()

	trie1.Put([]byte{1, 2, 3, 4, 5, 6}, []byte("world"))
	trie1.Put([]byte{1, 2, 3, 4}, []byte("hello"))

	trie2.Put([]byte{1, 2, 3, 4}, []byte("hello"))
	trie2.Put([]byte{1, 2, 3, 4, 5, 6}, []byte("world"))

	require.Equal(t, trie1.Hash(), trie2.Hash())
}

// Before put:
//
//  	           ┌───────────────────────────┐
//  	           │  Extension Node           │
//  	           │  Path: [0, 1, 0, 2, 0, 3] │
//  	           └────────────┬──────────────┘
//  	                        │
//  	┌───────────────────────┴──────────────────┐
//  	│                   Branch Node            │
//  	│   [0]         ...          [5]           │
//  	└────┼────────────────────────┼────────────┘
//  	     │                        │
//  	     │                        │
//  	     │                        │
//  	     │                        │
//   ┌───────┴──────────┐   ┌─────────┴─────────┐
//   │  Leaf Node       │   │  Leaf Node        │
//   │  Path: [4]       │   │  Path: [0]        │
//   │  Value: "hello1" │   │  Value: "hello2"  │
//   └──────────────────┘   └───────────────────┘
//
// After put([]byte{[1, 2, 3]}, "world"):
//  	           ┌───────────────────────────┐
//  	           │  Extension Node           │
//  	           │  Path: [0, 1, 0, 2, 0, 3] │
//  	           └────────────┬──────────────┘
//  	                        │
//  	┌───────────────────────┴────────────────────────┐
//  	│                   Branch Node                  │
//  	│   [0]         ...          [5]  value: "world" │
//  	└────┼────────────────────────┼──────────────────┘
//  	     │                        │
//  	     │                        │
//  	     │                        │
//  	     │                        │
//   ┌───────┴──────────┐   ┌─────────┴─────────┐
//   │  Leaf Node       │   │  Leaf Node        │
//   │  Path: [4]       │   │  Path: [0]        │
//   │  Value: "hello1" │   │  Value: "hello2"  │
//   └──────────────────┘   └───────────────────┘

func TestPutExtensionShorterAllMatched(t *testing.T) {
	trie := NewTrie()
	trie.Put([]byte{1, 2, 3, 4}, []byte("hello1"))
	trie.Put([]byte{1, 2, 3, 5}, []byte("hello2"))
	trie.Put([]byte{1, 2, 3}, []byte("world"))

	leaf1 := node.NewLeafNodeFromNibbles([]nibble.Nibble{}, []byte("hello1"))
	leaf2 := node.NewLeafNodeFromNibbles([]nibble.Nibble{}, []byte("hello2"))

	branch1 := node.NewBranchNode()
	branch1.SetBranch(nibble.Nibble(4), leaf1)
	branch1.SetBranch(nibble.Nibble(5), leaf2)

	branch2 := node.NewBranchNode()
	branch2.SetValue([]byte("world"))
	branch2.SetBranch(nibble.Nibble(0), branch1)

	ext := node.NewExtensionNode([]nibble.Nibble{0, 1, 0, 2, 0, 3}, branch2)

	require.Equal(t, ext.Hash(), trie.Hash())
}

func TestPutExtensionShorterPartialMatched(t *testing.T) {
	trie := NewTrie()
	trie.Put([]byte{1, 2, 3, 4}, []byte("hello1"))
	trie.Put([]byte{1, 2, 3, 5}, []byte("hello2"))
	trie.Put([]byte{1, 2, 5}, []byte("world"))

	leaf1 := node.NewLeafNodeFromNibbles([]nibble.Nibble{}, []byte("hello1"))
	leaf2 := node.NewLeafNodeFromNibbles([]nibble.Nibble{}, []byte("hello2"))

	branch1 := node.NewBranchNode()
	branch1.SetBranch(nibble.Nibble(4), leaf1)
	branch1.SetBranch(nibble.Nibble(5), leaf2)

	ext1 := node.NewExtensionNode([]nibble.Nibble{0}, branch1)

	branch2 := node.NewBranchNode()
	branch2.SetBranch(nibble.Nibble(3), ext1)
	leaf3 := node.NewLeafNodeFromNibbles([]nibble.Nibble{}, []byte("world"))
	branch2.SetBranch(nibble.Nibble(5), leaf3)

	ext2 := node.NewExtensionNode([]nibble.Nibble{0, 1, 0, 2, 0}, branch2)

	require.Equal(t, ext2.Hash(), trie.Hash())
}

func TestPutExtensionShorterZeroMatched(t *testing.T) {
	trie := NewTrie()
	trie.Put([]byte{1, 2, 3, 4}, []byte("hello1"))
	trie.Put([]byte{1, 2, 3, 5}, []byte("hello2"))
	trie.Put([]byte{1 << 4, 2, 5}, []byte("world"))

	leaf1 := node.NewLeafNodeFromNibbles([]nibble.Nibble{}, []byte("hello1"))
	leaf2 := node.NewLeafNodeFromNibbles([]nibble.Nibble{}, []byte("hello2"))

	branch1 := node.NewBranchNode()
	branch1.SetBranch(nibble.Nibble(4), leaf1)
	branch1.SetBranch(nibble.Nibble(5), leaf2)

	ext1 := node.NewExtensionNode([]nibble.Nibble{1, 0, 2, 0, 3, 0}, branch1)

	branch2 := node.NewBranchNode()
	branch2.SetBranch(nibble.Nibble(0), ext1)
	leaf3 := node.NewLeafNodeFromNibbles([]nibble.Nibble{0, 0, 2, 0, 5}, []byte("world"))
	branch2.SetBranch(nibble.Nibble(1), leaf3)

	require.Equal(t, branch2.Hash(), trie.Hash())
}

func TestPutExtensionAllMatched(t *testing.T) {
	trie := NewTrie()
	trie.Put([]byte{1, 2, 3, 4}, []byte("hello1"))
	trie.Put([]byte{1, 2, 3, 5 << 4}, []byte("hello2"))
	trie.Put([]byte{1, 2, 3}, []byte("world"))

	leaf1 := node.NewLeafNodeFromNibbles([]nibble.Nibble{4}, []byte("hello1"))
	leaf2 := node.NewLeafNodeFromNibbles([]nibble.Nibble{0}, []byte("hello2"))

	branch := node.NewBranchNode()
	branch.SetBranch(nibble.Nibble(0), leaf1)
	branch.SetBranch(nibble.Nibble(5), leaf2)
	branch.SetValue([]byte("world"))

	ext := node.NewExtensionNode([]nibble.Nibble{0, 1, 0, 2, 0, 3}, branch)

	require.Equal(t, ext.Hash(), trie.Hash())
}

func TestPutExtensionMore(t *testing.T) {
	trie := NewTrie()
	trie.Put([]byte{1, 2, 3, 4}, []byte("hello1"))
	trie.Put([]byte{1, 2, 3, 5}, []byte("hello2"))
	trie.Put([]byte{1, 2, 3, 6}, []byte("world"))

	leaf1 := node.NewLeafNodeFromNibbles([]nibble.Nibble{}, []byte("hello1"))
	leaf2 := node.NewLeafNodeFromNibbles([]nibble.Nibble{}, []byte("hello2"))
	leaf3 := node.NewLeafNodeFromNibbles([]nibble.Nibble{}, []byte("world"))

	branch := node.NewBranchNode()
	branch.SetBranch(nibble.Nibble(4), leaf1)
	branch.SetBranch(nibble.Nibble(5), leaf2)
	branch.SetBranch(nibble.Nibble(6), leaf3)

	ext := node.NewExtensionNode([]nibble.Nibble{0, 1, 0, 2, 0, 3, 0}, branch)

	require.Equal(t, ext.Hash(), trie.Hash())
}

func TestEthProof(t *testing.T) {
	mpt := new(trie.Trie)
	mpt.Update([]byte{1, 2, 3}, []byte("hello"))
	mpt.Update([]byte{1, 2, 3, 4, 5}, []byte("world"))
	w := proof.NewProofDB()
	err := mpt.Prove([]byte{1, 2, 3}, 0, w)
	require.NoError(t, err)
	rootHash := mpt.Hash()
	val, err := trie.VerifyProof(rootHash, []byte{1, 2, 3}, w)
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), val)
	fmt.Printf("root hash: %x\n", rootHash)
}

func TestMyTrie(t *testing.T) {
	tr := NewTrie()
	tr.Put([]byte{1, 2, 3}, []byte("hello"))
	tr.Put([]byte{1, 2, 3, 4, 5}, []byte("world"))
	n0, ok := tr.root.(*node.ExtensionNode)
	require.True(t, ok)
	n1, ok := n0.Next.(*node.BranchNode)
	require.True(t, ok)
	fmt.Printf("n0 hash: %x, Serialized: %x\n", n0.Hash(), n0.Serialize())
	fmt.Printf("n1 hash: %x, Serialized: %x\n", n1.Hash(), n1.Serialize())
}

func TestProveAndVerifyProof(t *testing.T) {
	t.Run("should not generate proof for non-exist key", func(t *testing.T) {
		tr := NewTrie()
		tr.Put([]byte{1, 2, 3}, []byte("hello"))
		tr.Put([]byte{1, 2, 3, 4, 5}, []byte("world"))
		notExistKey := []byte{1, 2, 3, 4}
		_, ok := tr.Prove(notExistKey)
		require.False(t, ok)
	})

	t.Run("should generate a proof for an existing key, the proof can be verified with the merkle root hash", func(t *testing.T) {
		tr := NewTrie()
		tr.Put([]byte{1, 2, 3}, []byte("hello"))
		tr.Put([]byte{1, 2, 3, 4, 5}, []byte("world"))

		key := []byte{1, 2, 3}
		proof, ok := tr.Prove(key)
		require.True(t, ok)

		rootHash := tr.Hash()

		// verify the proof with the root hash, the key in question and its proof
		val, err := VerifyProof(rootHash, key, proof)
		require.NoError(t, err)

		// when the verification has passed, it should return the correct value for the key
		require.Equal(t, []byte("hello"), val)
	})

	t.Run("should fail the verification of the trie was updated", func(t *testing.T) {
		tr := NewTrie()
		tr.Put([]byte{1, 2, 3}, []byte("hello"))
		tr.Put([]byte{1, 2, 3, 4, 5}, []byte("world"))

		// the hash was taken before the trie was updated
		rootHash := tr.Hash()

		// the proof was generated after the trie was updated
		tr.Put([]byte{5, 6, 7}, []byte("trie"))
		key := []byte{1, 2, 3}
		proof, ok := tr.Prove(key)
		require.True(t, ok)

		// should fail the verification since the merkle root hash doesn't match
		_, err := VerifyProof(rootHash, key, proof)
		require.Error(t, err)
	})
}
