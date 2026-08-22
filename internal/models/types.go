package models

import (
	"database/sql/driver"
	"encoding/json"
)

type JSONArray []string

func (j JSONArray) Value() (driver.Value, error) { return json.Marshal(j) }
func (j *JSONArray) Scan(value interface{}) error {
	if value == nil {
		*j = []string{}
		return nil
	}
	b, _ := value.([]byte)
	return json.Unmarshal(b, j)
}

type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) { return json.Marshal(j) }
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]interface{})
		return nil
	}
	b, _ := value.([]byte)
	return json.Unmarshal(b, j)
}
