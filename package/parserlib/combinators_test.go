package parserlib

import (
	"testing"
)

func TestCombinators(t *testing.T) {
	derps := Intercalate(
		&Keyword{Value: "derp"},
		&Keyword{Value: ","},
	)
	actual := derps.String()
	expected := `[",", "derp"] | "derp"`
	if expected != actual {
		t.Fatalf("expected %s; got %s", expected, actual)
	}
}
