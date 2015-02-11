package services

type Store interface {
	Set(key string, value string) error
	SetWithTTL(key string, value string, ttl uint64) error
	Get(key string) (string, error)
	GetRecursive(prefix string) ([]Node, error)
}

type Node struct {
	Key   string
	Value string
}
