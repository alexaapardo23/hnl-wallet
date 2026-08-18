package seed

import (
	"encoding/json"
	"fmt"
	"os"

	"hnl-wallet/backend/models"
)

func LoadData(path string) (*models.Data, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open data file: %w", err)
	}
	defer file.Close()

	var data models.Data

	decoder := json.NewDecoder(file)

	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode data file: %w", err)
	}

	return &data, nil
}