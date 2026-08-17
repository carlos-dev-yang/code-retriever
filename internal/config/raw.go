package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"
)

type RawConfig struct {
	Version   int          `json:"version"`
	Index     RawIndex     `json:"index"`
	Embedding RawEmbedding `json:"embedding"`
	Search    RawSearch    `json:"search"`
	MCP       RawMCP       `json:"mcp"`
}

type RawIndex struct {
	Languages          []string `json:"languages"`
	MaxSourceFileBytes int      `json:"max_source_file_bytes,omitempty"`
	TargetSegmentBytes int      `json:"target_segment_bytes,omitempty"`
}

type RawEmbedding struct {
	Model             string     `json:"model"`
	ServingDimensions *int       `json:"serving_dimensions"`
	Reducer           string     `json:"reducer"`
	Normalizer        string     `json:"normalizer"`
	Metric            string     `json:"metric"`
	StorageCodec      *string    `json:"storage_codec,omitempty"`
	Request           RawRequest `json:"request"`
	Retry             RawRetry   `json:"retry"`
}

type RawRequest struct {
	MaxInputs          int `json:"max_inputs,omitempty"`
	MaxTotalInputBytes int `json:"max_total_input_bytes,omitempty"`
	MaxConcurrency     int `json:"max_concurrency,omitempty"`
	TimeoutSeconds     int `json:"timeout_seconds,omitempty"`
}

type RawRetry struct {
	MaxRetries  *int   `json:"max_retries,omitempty"`
	WaitSeconds *[]int `json:"wait_seconds,omitempty"`
}

type RawSearch struct {
	DefaultMode             *string       `json:"default_mode"`
	AllowPaidQueryEmbedding *bool         `json:"allow_paid_query_embedding"`
	ReturnK                 *int          `json:"return_k"`
	CandidateK              *int          `json:"candidate_k"`
	RRFK                    *int          `json:"rrf_k"`
	MaxQueryBytes           *int          `json:"max_query_bytes"`
	MaxQueryTokens          *int          `json:"max_query_tokens"`
	MaxQueryTokenRunes      *int          `json:"max_query_token_runes"`
	FTSWeights              RawFTSWeights `json:"fts_weights"`
}

type RawFTSWeights struct {
	Symbols *float64 `json:"symbols"`
	Body    *float64 `json:"body"`
}
type RawMCP struct {
	HardMaxInlineBytes int `json:"hard_max_inline_bytes,omitempty"`
}

func DecodeRaw(data []byte) (RawConfig, error) {
	if !utf8.Valid(data) {
		return RawConfig{}, fmt.Errorf("config is not valid UTF-8")
	}
	if err := rejectDuplicateKeys(data); err != nil {
		return RawConfig{}, err
	}
	if err := detectLegacyFields(data); err != nil {
		return RawConfig{}, err
	}
	if err := rejectExplicitZeroSafetyValues(data); err != nil {
		return RawConfig{}, err
	}
	if err := rejectExplicitRetryWaits(data); err != nil {
		return RawConfig{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var raw RawConfig
	if err := decoder.Decode(&raw); err != nil {
		return RawConfig{}, err
	}
	if token, err := decoder.Token(); err != io.EOF || token != nil {
		return RawConfig{}, fmt.Errorf("config contains trailing JSON value")
	}
	return raw, nil
}

func rejectExplicitZeroSafetyValues(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	root, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	if mcp, specified := root["mcp"]; specified && mcp == nil {
		return fmt.Errorf("mcp must not be null when specified")
	}
	if embedding, specified := root["embedding"]; specified && embedding == nil {
		return fmt.Errorf("embedding must not be null when specified")
	}
	for _, path := range [][2]string{
		{"index", "max_source_file_bytes"},
		{"index", "target_segment_bytes"},
		{"mcp", "hard_max_inline_bytes"},
	} {
		if explicitNull(root, path[0], path[1]) {
			return fmt.Errorf("%s.%s must not be null when specified", path[0], path[1])
		}
		if explicitZero(root, path[0], path[1]) {
			return fmt.Errorf("%s.%s must be positive when specified", path[0], path[1])
		}
	}
	if embedding, ok := root["embedding"].(map[string]any); ok {
		for _, field := range []string{"serving_dimensions", "storage_codec"} {
			if embedding[field] == nil {
				if _, specified := embedding[field]; specified {
					return fmt.Errorf("embedding.%s must not be null when specified", field)
				}
			}
		}
		requestValue, requestSpecified := embedding["request"]
		if requestSpecified && requestValue == nil {
			return fmt.Errorf("embedding.request must not be null when specified")
		}
		if request, ok := requestValue.(map[string]any); ok {
			for _, field := range []string{"max_inputs", "max_total_input_bytes", "max_concurrency", "timeout_seconds"} {
				if request[field] == nil {
					if _, specified := request[field]; specified {
						return fmt.Errorf("embedding.request.%s must not be null when specified", field)
					}
				}
				if number, ok := request[field].(json.Number); ok && jsonNumberIsZero(number) {
					return fmt.Errorf("embedding.request.%s must be positive when specified", field)
				}
			}
		}
		retryValue, retrySpecified := embedding["retry"]
		if retrySpecified && retryValue == nil {
			return fmt.Errorf("embedding.retry must not be null when specified")
		}
		if retry, ok := retryValue.(map[string]any); ok {
			if retry["max_retries"] == nil {
				if _, specified := retry["max_retries"]; specified {
					return fmt.Errorf("embedding.retry.max_retries must not be null when specified")
				}
			}
		}
	}
	return nil
}

func explicitZero(root map[string]any, object, field string) bool {
	value, ok := root[object].(map[string]any)
	if !ok {
		return false
	}
	number, ok := value[field].(json.Number)
	return ok && jsonNumberIsZero(number)
}

func explicitNull(root map[string]any, object, field string) bool {
	value, ok := root[object].(map[string]any)
	if !ok {
		return false
	}
	fieldValue, exists := value[field]
	return exists && fieldValue == nil
}

func jsonNumberIsZero(number json.Number) bool {
	value, err := strconv.ParseInt(number.String(), 10, 64)
	return err == nil && value == 0
}

func rejectExplicitRetryWaits(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	root, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	embedding, ok := root["embedding"].(map[string]any)
	if !ok {
		return nil
	}
	retry, ok := embedding["retry"].(map[string]any)
	if !ok {
		return nil
	}
	waits, exists := retry["wait_seconds"]
	if !exists {
		return nil
	}
	values, ok := waits.([]any)
	if !ok || len(values) == 0 {
		return fmt.Errorf("embedding.retry.wait_seconds must be a non-empty array when specified")
	}
	return nil
}

func rejectDuplicateKeys(data []byte) error {
	if err := rejectLoneSurrogateEscapes(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	var walk func() error
	walk = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, isDelimiter := token.(json.Delim)
		if !isDelimiter {
			return nil
		}
		switch delimiter {
		case '{':
			keys := map[string]struct{}{}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key := keyToken.(string)
				if _, exists := keys[key]; exists {
					return fmt.Errorf("duplicate config field %q", key)
				}
				keys[key] = struct{}{}
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		default:
			return fmt.Errorf("invalid JSON delimiter")
		}
	}
	if err := walk(); err != nil {
		return err
	}
	_, err := decoder.Token()
	if err != io.EOF {
		return fmt.Errorf("config contains trailing data")
	}
	return nil
}

func rejectLoneSurrogateEscapes(data []byte) error {
	inString := false
	for index := 0; index < len(data); index++ {
		if !inString {
			if data[index] == '"' {
				inString = true
			}
			continue
		}
		if data[index] == '"' {
			inString = false
			continue
		}
		if data[index] != '\\' {
			continue
		}
		if index+1 >= len(data) {
			return fmt.Errorf("invalid string escape")
		}
		if data[index+1] != 'u' {
			index++
			continue
		}
		if index+5 >= len(data) {
			return fmt.Errorf("invalid unicode escape")
		}
		unit, ok := hexUnit(data[index+2 : index+6])
		if !ok {
			return fmt.Errorf("invalid unicode escape")
		}
		if unit >= 0xd800 && unit <= 0xdbff {
			if index+11 >= len(data) || data[index+6] != '\\' || data[index+7] != 'u' {
				return fmt.Errorf("lone UTF-16 surrogate escape")
			}
			low, ok := hexUnit(data[index+8 : index+12])
			if !ok || low < 0xdc00 || low > 0xdfff {
				return fmt.Errorf("lone UTF-16 surrogate escape")
			}
			index += 11
			continue
		}
		if unit >= 0xdc00 && unit <= 0xdfff {
			return fmt.Errorf("lone UTF-16 surrogate escape")
		}
		index += 5
	}
	return nil
}

func hexUnit(bytes []byte) (rune, bool) {
	var value rune
	for _, byteValue := range bytes {
		value <<= 4
		switch {
		case byteValue >= '0' && byteValue <= '9':
			value += rune(byteValue - '0')
		case byteValue >= 'a' && byteValue <= 'f':
			value += rune(byteValue-'a') + 10
		case byteValue >= 'A' && byteValue <= 'F':
			value += rune(byteValue-'A') + 10
		default:
			return 0, false
		}
	}
	return value, true
}
