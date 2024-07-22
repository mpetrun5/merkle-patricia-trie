package node

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mpetrun5/merkle-patrica-trie/nibble"
)

type ExtensionNode struct {
	Path []nibble.Nibble
	Next Node
}

func NewExtensionNode(nibbles []nibble.Nibble, next Node) *ExtensionNode {
	return &ExtensionNode{
		Path: nibbles,
		Next: next,
	}
}

func (e ExtensionNode) Hash() []byte {
	return crypto.Keccak256(e.Serialize())
}

func (e ExtensionNode) Raw() []interface{} {
	hashes := make([]interface{}, 2)
	hashes[0] = nibble.ToBytes(nibble.ToPrefixed(e.Path, false))
	if len(Serialize(e.Next)) >= 32 {
		hashes[1] = e.Next.Hash()
	} else {
		hashes[1] = e.Next.Raw()
	}
	return hashes
}

func (e ExtensionNode) Serialize() []byte {
	return Serialize(e)
}
