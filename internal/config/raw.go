package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	Languages            []string `json:"languages"`
	MaxSourceFileBytes   int      `json:"max_source_file_bytes"`
	MaxChunkBytes        int      `json:"max_chunk_bytes"`
	MaxSegmentInputBytes int      `json:"max_segment_input_bytes"`
}

type RawEmbedding struct {
	Model            string   `json:"model"`
	TargetDimensions *int     `json:"target_dimensions"`
	Reducer          string   `json:"reducer"`
	Normalizer       string   `json:"normalizer"`
	Metric           string   `json:"metric"`
	StorageCodec     *string  `json:"storage_codec"`
	Batch            RawBatch `json:"batch"`
}

type RawBatch struct {
	MaxInputs        int `json:"max_inputs"`
	MaxInputTokens   int `json:"max_input_tokens"`
	MaxRetries       int `json:"max_retries"`
	RequestTimeoutMS int `json:"request_timeout_ms"`
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
	HardMaxInlineBytes int `json:"hard_max_inline_bytes"`
	MaxReadSpanLines   int `json:"max_read_span_lines"`
}

func DecodeRaw(data []byte) (RawConfig, error) {
	if !utf8.Valid(data) {
		return RawConfig{}, fmt.Errorf("config is not valid UTF-8")
	}
	if err := rejectDuplicateKeys(data); err != nil {
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
