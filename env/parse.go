package env

import (
	"encoding/json"
	"os"
)

// ParseJSON parses the variable contents as JSON
func ParseJSON(varName string, v interface{}) error {
	return json.Unmarshal([]byte(os.Getenv(varName)), v)
}
