package repository

type (
	DB interface {
		Connect() bool
	}
)
