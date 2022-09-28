package common

import "github.com/Connect-Club/connectclub-mobile-common/storage"

type PublicStorage interface {
	storage.Storage
}

func SetStorage(s PublicStorage) {
	storage.Set(s)
}
