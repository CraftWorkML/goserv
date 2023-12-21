package repository

import (
	"fmt"
	cfg "goserv/src/configuration"
	"log"
)

type (
	AuthDB interface {
		UploadUser(accessToken, refreshToken string) error
		VerifyUser(accessToken string) bool
		Connect() bool
	}
	InMemoryDB struct {
		table map[string]string
	}
)

func NewAuthDataBase(config *cfg.Properties) (AuthDB, error) {
	if config == nil {
		return nil, fmt.Errorf("config is not valid")
	}
	return &InMemoryDB{}, nil
}

func (i *InMemoryDB) Connect() bool {
	if i.table == nil {
		i.table = make(map[string]string)
	}
	return true
}

func (i *InMemoryDB) UploadUser(accessToken, refreshToken string) error {
	if i.table == nil {
		return fmt.Errorf("can not uploat user, connection is off")
	}
	i.table[accessToken] = refreshToken
	log.Printf("uploaded an user %v to db %v", accessToken, i.table)
	return nil
}

func (i *InMemoryDB) VerifyUser(accessToken string) bool {
	if i.table == nil {
		return false
	}
	token, ok := i.table[accessToken]
	if !ok {
		return false
	}
	fmt.Printf("add verify token %v", token)
	return true
}
