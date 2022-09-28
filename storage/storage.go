package storage

var _storage Storage

type Storage interface {
	GetString(key string) string
	SetString(key string, value string)
	Delete(key string)
}

func Set(s Storage) {
	_storage = s
}

func Get() Storage {
	return _storage
}
