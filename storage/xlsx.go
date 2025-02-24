package storage

import (
	"context"
	"fmt"
	"os"
)

type XLSX struct {
}

func NewXLSXClient() *XLSX {
	return &XLSX{}
}

func (x *XLSX) Save(ctx context.Context, data []byte, fileName string) error {
	err := os.WriteFile(fmt.Sprintf("%s.json", fileName), data, 0644)
	if err != nil {
		return err
	}

	return nil
}
