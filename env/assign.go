// Package env contains functions to manage and use env vars
package env

// AssignString assigns the env var as a string to the given pointer.
func AssignString(p *string, varName string) {
	*p = GetString(varName, *p)
}

// AssignBool assigns the env var as a boolean to the given pointer.
func AssignBool(p *bool, varName string) {
	*p = GetBool(varName, *p)
}

// AssignInt64 assigns the env var as an int to the given pointer.
func AssignInt64(p *int64, varName string) {
	*p = GetInt64(varName, *p)
}

// AssignFloat64 assigns the env var a float to the given pointer.
func AssignFloat64(p *float64, varName string) {
	*p = GetFloat64(varName, *p)
}

// AssignStringArray assigns the env var a []string to the given pointer.
func AssignStringArray(p *[]string, varName string, sep string) {
	*p = GetStringArray(varName, sep, *p)
}

// AssignStringSet assigns the env var a StringSet to the given pointer.
func AssignStringSet(p *StringSet, varName string, sep string) {
	*p = GetStringSet(varName, sep, *p)
}
