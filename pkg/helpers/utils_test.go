package helpers

import (
	"testing"
)

func TestGetInt(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]interface{}
		key      string
		def      int
		expected int
	}{
		{
			name: "valid string number",
			cfg:  map[string]interface{}{"port": "8080"},
			key:  "port",
			def:  80,
			expected: 8080,
		},
		{
			name: "invalid string number",
			cfg:  map[string]interface{}{"port": "invalid"},
			key:  "port",
			def:  80,
			expected: 80,
		},
		{
			name: "missing key",
			cfg:  map[string]interface{}{},
			key:  "port",
			def:  80,
			expected: 80,
		},
		{
			name: "non-string value",
			cfg:  map[string]interface{}{"port": 8080},
			key:  "port",
			def:  80,
			expected: 80,
		},
		{
			name: "zero value",
			cfg:  map[string]interface{}{"port": "0"},
			key:  "port",
			def:  80,
			expected: 0,
		},
		{
			name: "negative value",
			cfg:  map[string]interface{}{"port": "-1"},
			key:  "port",
			def:  80,
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetInt(tt.cfg, tt.key, tt.def)
			if result != tt.expected {
				t.Errorf("GetInt() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetString(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]interface{}
		key      string
		expected string
	}{
		{
			name: "valid string",
			cfg:  map[string]interface{}{"url": "localhost"},
			key:  "url",
			expected: "localhost",
		},
		{
			name: "empty string",
			cfg:  map[string]interface{}{"url": ""},
			key:  "url",
			expected: "",
		},
		{
			name: "missing key",
			cfg:  map[string]interface{}{},
			key:  "url",
			expected: "",
		},
		{
			name: "non-string value",
			cfg:  map[string]interface{}{"url": 123},
			key:  "url",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetString(tt.cfg, tt.key)
			if result != tt.expected {
				t.Errorf("GetString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetBoolWithDefault(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]interface{}
		key      string
		def      bool
		expected bool
	}{
		{
			name: "valid true bool",
			cfg:  map[string]interface{}{"enabled": true},
			key:  "enabled",
			def:  false,
			expected: true,
		},
		{
			name: "valid false bool",
			cfg:  map[string]interface{}{"enabled": false},
			key:  "enabled",
			def:  true,
			expected: false,
		},
		{
			name: "missing key uses default",
			cfg:  map[string]interface{}{},
			key:  "enabled",
			def:  true,
			expected: true,
		},
		{
			name: "non-bool value uses default",
			cfg:  map[string]interface{}{"enabled": "true"},
			key:  "enabled",
			def:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBoolWithDefault(tt.cfg, tt.key, tt.def)
			if result != tt.expected {
				t.Errorf("GetBoolWithDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetIntWithDefault(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]interface{}
		key      string
		def      int
		expected int
	}{
		{
			name: "valid string number",
			cfg:  map[string]interface{}{"timeout": "30"},
			key:  "timeout",
			def:  10,
			expected: 30,
		},
		{
			name: "invalid string number uses default",
			cfg:  map[string]interface{}{"timeout": "invalid"},
			key:  "timeout",
			def:  10,
			expected: 10,
		},
		{
			name: "missing key uses default",
			cfg:  map[string]interface{}{},
			key:  "timeout",
			def:  10,
			expected: 10,
		},
		{
			name: "non-string value uses default",
			cfg:  map[string]interface{}{"timeout": 30},
			key:  "timeout",
			def:  10,
			expected: 10,
		},
		{
			name: "zero value",
			cfg:  map[string]interface{}{"timeout": "0"},
			key:  "timeout",
			def:  10,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetIntWithDefault(tt.cfg, tt.key, tt.def)
			if result != tt.expected {
				t.Errorf("GetIntWithDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]interface{}
		key      string
		expected []string
	}{
		{
			name: "comma separated values",
			cfg:  map[string]interface{}{"urls": "server1,server2,server3"},
			key:  "urls",
			expected: []string{"server1", "server2", "server3"},
		},
		{
			name: "single value",
			cfg:  map[string]interface{}{"urls": "server1"},
			key:  "urls",
			expected: []string{"server1"},
		},
		{
			name: "values with spaces",
			cfg:  map[string]interface{}{"urls": " server1 , server2 , server3 "},
			key:  "urls",
			expected: []string{"server1", "server2", "server3"},
		},
		{
			name: "empty string",
			cfg:  map[string]interface{}{"urls": ""},
			key:  "urls",
			expected: []string{""},
		},
		{
			name: "missing key",
			cfg:  map[string]interface{}{},
			key:  "urls",
			expected: []string{},
		},
		{
			name: "non-string value",
			cfg:  map[string]interface{}{"urls": []string{"server1", "server2"}},
			key:  "urls",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetStringSlice(tt.cfg, tt.key)
			if len(result) != len(tt.expected) {
				t.Errorf("GetStringSlice() length = %v, want %v", len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("GetStringSlice()[%d] = %v, want %v", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestGetBoolSlice(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]interface{}
		key      string
		expected []bool
	}{
		{
			name: "comma separated boolean values",
			cfg:  map[string]interface{}{"flags": "true,false,true"},
			key:  "flags",
			expected: []bool{true, false, true},
		},
		{
			name: "single boolean value",
			cfg:  map[string]interface{}{"flags": "true"},
			key:  "flags",
			expected: []bool{true},
		},
		{
			name: "values with spaces",
			cfg:  map[string]interface{}{"flags": " true , false , true "},
			key:  "flags",
			expected: []bool{true, false, true},
		},
		{
			name: "non-true values are false",
			cfg:  map[string]interface{}{"flags": "yes,no,1,0,false"},
			key:  "flags",
			expected: []bool{false, false, false, false, false},
		},
		{
			name: "missing key",
			cfg:  map[string]interface{}{},
			key:  "flags",
			expected: []bool{},
		},
		{
			name: "non-string value",
			cfg:  map[string]interface{}{"flags": []bool{true, false}},
			key:  "flags",
			expected: []bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBoolSlice(tt.cfg, tt.key)
			if len(result) != len(tt.expected) {
				t.Errorf("GetBoolSlice() length = %v, want %v", len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("GetBoolSlice()[%d] = %v, want %v", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]interface{}
		key      string
		def      bool
		expected bool
	}{
		{
			name: "valid true bool",
			cfg:  map[string]interface{}{"enabled": true},
			key:  "enabled",
			def:  false,
			expected: true,
		},
		{
			name: "valid false bool",
			cfg:  map[string]interface{}{"enabled": false},
			key:  "enabled",
			def:  true,
			expected: false,
		},
		{
			name: "missing key uses default",
			cfg:  map[string]interface{}{},
			key:  "enabled",
			def:  true,
			expected: true,
		},
		{
			name: "non-bool value uses default",
			cfg:  map[string]interface{}{"enabled": "true"},
			key:  "enabled",
			def:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBool(tt.cfg, tt.key, tt.def)
			if result != tt.expected {
				t.Errorf("GetBool() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseEndOfRecordChar(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]interface{}
		expected byte
	}{
		{
			name: "EOT end of record",
			cfg:  map[string]interface{}{"endOfRecordChar": "EOT"},
			expected: 0x04,
		},
		{
			name: "LEN end of record",
			cfg:  map[string]interface{}{"endOfRecordChar": "LEN"},
			expected: 0x00,
		},
		{
			name: "default ETX (case insensitive)",
			cfg:  map[string]interface{}{"endOfRecordChar": "etx"},
			expected: 0x03,
		},
		{
			name: "unknown value defaults to ETX",
			cfg:  map[string]interface{}{"endOfRecordChar": "unknown"},
			expected: 0x03,
		},
		{
			name: "missing key defaults to ETX",
			cfg:  map[string]interface{}{},
			expected: 0x03,
		},
		{
			name: "case insensitive EOT",
			cfg:  map[string]interface{}{"endOfRecordChar": "eot"},
			expected: 0x04,
		},
		{
			name: "case insensitive LEN",
			cfg:  map[string]interface{}{"endOfRecordChar": "len"},
			expected: 0x00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseEndOfRecordChar(tt.cfg)
			if result != tt.expected {
				t.Errorf("ParseEndOfRecordChar() = 0x%02x, want 0x%02x", result, tt.expected)
			}
		})
	}
}

func TestParseMessageLenType(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]interface{}
		expected int
	}{
		{
			name: "EXCLUDELEN message type",
			cfg:  map[string]interface{}{"messageLenType": "EXCLUDELEN"},
			expected: 1,
		},
		{
			name: "case insensitive EXCLUDELEN",
			cfg:  map[string]interface{}{"messageLenType": "excludelen"},
			expected: 1,
		},
		{
			name: "INCLUDELEN message type",
			cfg:  map[string]interface{}{"messageLenType": "INCLUDELEN"},
			expected: 0,
		},
		{
			name: "unknown value defaults to INCLUDELEN",
			cfg:  map[string]interface{}{"messageLenType": "unknown"},
			expected: 0,
		},
		{
			name: "missing key defaults to INCLUDELEN",
			cfg:  map[string]interface{}{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseMessageLenType(tt.cfg)
			if result != tt.expected {
				t.Errorf("ParseMessageLenType() = %v, want %v", result, tt.expected)
			}
		})
	}
}