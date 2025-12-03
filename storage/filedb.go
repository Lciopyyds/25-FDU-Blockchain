package storage

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"

	"mychain/core"
)

// FileStorage 负责把区块链存到一个 JSON 文件中
type FileStorage struct {
	Path string
}

// 创建一个新的文件存储
func NewFileStorage(path string) *FileStorage {
	return &FileStorage{Path: path}
}

// Save 把整个区块链写入到文件
func (fs *FileStorage) Save(bc *core.Blockchain) error {
	data, err := json.MarshalIndent(bc, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(fs.Path, data, 0644)
}

// Load 从文件中读取区块链
func (fs *FileStorage) Load() (*core.Blockchain, error) {
	// 检查文件是否存在
	_, err := os.Stat(fs.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err // 调用方用 os.IsNotExist 判断
		}
		return nil, err
	}

	data, err := ioutil.ReadFile(fs.Path)
	if err != nil {
		return nil, err
	}

	var bc core.Blockchain
	if err := json.Unmarshal(data, &bc); err != nil {
		return nil, err
	}

	// 如果文件里 blocks 是空的，视为错误
	if len(bc.Blocks) == 0 {
		return nil, errors.New("loaded blockchain has no blocks")
	}

	return &bc, nil
}
