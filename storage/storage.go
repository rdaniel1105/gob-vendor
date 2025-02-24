package storage

import "context"

type Storage interface {
	Save(context.Context, []byte) error
}
