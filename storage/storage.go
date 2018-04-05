package storage

func NewStorage(filename string) (Storage, error) {
	storage, err := NewLeveldbStorage(filename)
	if err != nil {
		return nil, err
	}
	return storage, err
}
