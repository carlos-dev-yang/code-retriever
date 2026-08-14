package lexical

import (
	"errors"
	"reflect"
	"testing"

	"cidx/internal/config"
	"cidx/internal/symbol"
)

var queryLimits = config.QueryLimits{MaxBytes: config.DefaultMaxQueryBytes, MaxTokens: config.DefaultMaxQueryTokens, MaxTokenRunes: config.DefaultMaxQueryTokenRunes}

func TestBuildQueryUsesSharedIdentifierNormalizerAndQuotedGrammar(t *testing.T) {
	query, err := BuildQuery("GetUserByID", symbol.IdentifierNormalizer{}, queryLimits)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := query.IdentifierTokens, []string{"get", "user", "by", "id"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("identifier tokens=%#v", got)
	}
	if query.ExactSymbolCandidate != "get user by id" {
		t.Fatalf("exact candidate=%q", query.ExactSymbolCandidate)
	}
	if query.MatchExpression != `"get" AND "user" AND "by" AND "id"` {
		t.Fatalf("MATCH=%q", query.MatchExpression)
	}

	injection, err := BuildQuery(`alpha" OR body:* (NEAR)`, symbol.IdentifierNormalizer{}, queryLimits)
	if err != nil {
		t.Fatal(err)
	}
	if injection.MatchExpression != `"or" AND "near" AND "alpha" AND "body"` {
		t.Fatalf("unsafe input expression=%q", injection.MatchExpression)
	}
}

func TestBuildQueryRejectsEmptyAndInvalidUTF8(t *testing.T) {
	for _, input := range []string{"", " \t()!*:\n"} {
		_, err := BuildQuery(input, symbol.IdentifierNormalizer{}, queryLimits)
		var queryError *QueryError
		if !errors.As(err, &queryError) || queryError.Code != EmptyQuery {
			t.Fatalf("%q error=%v", input, err)
		}
	}
	_, err := BuildQuery(string([]byte{0xff}), symbol.IdentifierNormalizer{}, queryLimits)
	var queryError *QueryError
	if !errors.As(err, &queryError) || queryError.Code != InvalidQuery {
		t.Fatalf("invalid UTF-8 error=%v", err)
	}
}

func TestBuildQueryUsesResolvedLimits(t *testing.T) {
	for _, limits := range []config.QueryLimits{
		{MaxBytes: 3, MaxTokens: 4, MaxTokenRunes: 4},
		{MaxBytes: 32, MaxTokens: 1, MaxTokenRunes: 16},
		{MaxBytes: 32, MaxTokens: 4, MaxTokenRunes: 3},
	} {
		if _, err := BuildQuery("four tokens here now", symbol.IdentifierNormalizer{}, limits); err == nil {
			t.Fatalf("limits %+v accepted excessive query", limits)
		}
	}
}
