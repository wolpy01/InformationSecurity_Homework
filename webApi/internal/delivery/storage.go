package delivery

import "proxyServer/mongo/domain"

type Storage interface {
	GetAll() ([]domain.HTTPTransaction, error)
	GetByID(string) (domain.HTTPTransaction, error)
}
