package typecheck

type Fact struct {
	Subject Hash
	Attr    Attr
	Object  Hash
}

type Attr string

type Query struct {
	Subject Hash
	Object  Hash
}

type Store interface {
	Put(Fact)
	Get(Query)
}

type Typechecker struct {
	store Store
}
