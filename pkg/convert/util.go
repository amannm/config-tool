package convert

import (
	"os"
	"path"
	"strings"
)

func ReadAllFiles[T any](folderPath string, fileSuffix string, handler func([]byte) (*T, error)) (*T, error) {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, fileSuffix) {
			filePath := path.Join(folderPath, name)
			data, err := os.ReadFile(filePath)
			if err != nil {
				return nil, err
			}
			return handler(data)
		}
	}
	return nil, nil
}
