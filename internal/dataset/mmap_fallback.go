//go:build !unix

package dataset

import (
	"fmt"
	"os"
)

func mmapFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read references file: %w", err)
	}

	return data, nil
}

func unmapFile(_ []byte) error {
	return nil
}
