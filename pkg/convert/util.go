package convert

import (
	"os"
	"path"
	"strings"
)

func ReadAllFiles(folderPath string, fileSuffix string, handler func([]byte) error) error {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, fileSuffix) {
			filePath := path.Join(folderPath, name)
			data, err := os.ReadFile(filePath)
			if err != nil {
				return err
			}
			err = handler(data)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
