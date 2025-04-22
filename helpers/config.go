package helpers

import (
	"log"
	"strconv"
	"strings"
)

func getString(cfg map[string]interface{}, key string) string {
	if v, ok := cfg[key].(string); ok {
		return v
	}
	log.Printf("%s not provided or not a string", key)
	return ""
}

func getBoolWithDefault(cfg map[string]interface{}, key string, def bool) bool {
	if v, ok := cfg[key].(bool); ok {
		return v
	}
	log.Printf("%s not provided or not a bool - default to %v", key, def)
	return def
}

func getIntWithDefault(cfg map[string]interface{}, key string, def int) int {
	var err error
	var i int
	if v, ok := cfg[key].(string); ok {
		if i, err = strconv.Atoi(v); err == nil {
			return i
		}
		log.Printf("Invalid integer for %s: %v - default to %d", key, err, def)
	}
	log.Printf("%s not provided or not a string - default to %d", key, def)
	return def
}

func getStringSlice(cfg map[string]interface{}, key string) []string {
	if v, ok := cfg[key].(string); ok {
		parts := strings.Split(v, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts
	}
	log.Printf("%s not provided or not a string", key)
	return []string{}
}

func getBoolSlice(cfg map[string]interface{}, key string) []bool {
	if v, ok := cfg[key].(string); ok {
		parts := strings.Split(v, ",")
		bools := make([]bool, len(parts))
		for i, s := range parts {
			bools[i] = strings.TrimSpace(s) == "true"
		}
		return bools
	}
	log.Printf("%s not provided or not a string", key)
	return []bool{}
}

