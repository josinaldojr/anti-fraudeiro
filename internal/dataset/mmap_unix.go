//go:build unix

package dataset

import (
	"fmt"
	"os"
	"syscall"
)

func mmapFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open references file for mmap: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat references file for mmap: %w", err)
	}

	if info.Size() == 0 {
		return nil, fmt.Errorf("references file is empty")
	}

	data, err := syscall.Mmap(int(file.Fd()), 0, int(info.Size()), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, fmt.Errorf("mmap references file: %w", err)
	}

	return data, nil
}

func unmapFile(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	return syscall.Munmap(data)
}
