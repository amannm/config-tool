package convert

import (
	"os"
	"path"
	"strings"
)

func ReadAllFiles(folderPath string, fileSuffix string) ([][]byte, error) {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return nil, err
	}
	fileContents := [][]byte{}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, fileSuffix) {
			filePath := path.Join(folderPath, name)
			data, err := os.ReadFile(filePath)
			if err != nil {
				return nil, err
			}
			fileContents = append(fileContents, data)
		}
	}
	return fileContents, nil
}

func WriteFile(content []byte, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = f.Write(content)
	if err != nil {
		return err
	}
	return nil
}
