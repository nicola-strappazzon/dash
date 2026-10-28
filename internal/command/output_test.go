package command

import (
	"context"
	"encoding/json"
	"math"
	"testing"

	"github.com/nicola-strappazzon/go-table"
)

func TestJSONFlagValue(t *testing.T) {
	enabled := false
	ctx := WithJSON(context.Background(), &enabled)
	if JSON(ctx) || JSON(context.Background()) {
		t.Fatal("JSON should default to false")
	}
	enabled = true
	if !JSON(ctx) {
		t.Fatal("context did not observe parsed flag")
	}
}

func TestPrintSerializationError(t *testing.T) {
	enabled := true
	err := Print(WithJSON(context.Background(), &enabled), table.New().Add(math.NaN()))
	if _, ok := err.(*json.UnsupportedValueError); !ok {
		t.Fatalf("serialization error was not propagated: %v", err)
	}
}
