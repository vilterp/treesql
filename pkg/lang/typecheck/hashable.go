package typecheck

type Hash uint64

type Hashable interface {
	GetHash() Hash
}

type ContentAddressableStore interface {
	Put(h Hashable) Hash
	Get(h Hash) Hashable
}
