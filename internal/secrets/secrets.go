package secrets

import "github.com/zalando/go-keyring"

const service = "clickwheel"

type Store struct{}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Set(key, value string) error {
	return keyring.Set(service, key, value)
}

func (s *Store) Get(key string) (string, error) {
	return keyring.Get(service, key)
}

func (s *Store) Delete(key string) error {
	return keyring.Delete(service, key)
}
