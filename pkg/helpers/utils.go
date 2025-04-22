package helpers

import (
	"log"
	"strconv"
	"strings"
)
func GetInt(cfg map[string]interface{}, key string, def int) int {
	if val, ok := cfg[key].(string); ok {
		if num, err := strconv.Atoi(val); err == nil {
			return num
		}
		log.Printf("Invalid int for %s: %v", key, val)
	}
	return def
}

func GetString(cfg map[string]interface{}, key string) string {
	if v, ok := cfg[key].(string); ok {
		return v
	}
	log.Printf("%s not provided or not a string", key)
	return ""
}

func GetBoolWithDefault(cfg map[string]interface{}, key string, def bool) bool {
	if v, ok := cfg[key].(bool); ok {
		return v
	}
	log.Printf("%s not provided or not a bool - default to %v", key, def)
	return def
}

func GetIntWithDefault(cfg map[string]interface{}, key string, def int) int {
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

func GetStringSlice(cfg map[string]interface{}, key string) []string {
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

func GetBoolSlice(cfg map[string]interface{}, key string) []bool {
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



func GetBool(cfg map[string]interface{}, key string, def bool) bool {
	if val, ok := cfg[key].(bool); ok {
		return val
	}
	log.Printf("%s not provided or not a bool", key)
	return def
}
func ParseEndOfRecordChar(cfg map[string]interface{}) byte {
	val := GetString(cfg, "endOfRecordChar")
	switch strings.ToUpper(val) {
	case "EOT":
		return 0x04
	case "LEN":
		return 0x00
	default:
		return 0x03
	}
}

func ParseMessageLenType(cfg map[string]interface{}) int {
	val := GetString(cfg, "messageLenType")
	switch strings.ToUpper(val) {
	case "EXCLUDELEN":
		return 1
	default:
		return 0 // default is INCLUDELEN
	}
}

