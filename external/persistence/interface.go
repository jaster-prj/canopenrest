package persistence

type IPersistence interface {
	SafeNode(id int, odsFile []byte) error
	GetNodes() ([]int, error)
	GetObjDict(id int) ([]byte, error)
}
