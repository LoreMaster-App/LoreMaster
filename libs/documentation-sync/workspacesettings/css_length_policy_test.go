package workspacesettings

import (
	"encoding/json"
	"testing"
)

func TestNormalizeCSSLengthTurnsABareNumberIntoPixels(t *testing.T) {
	for in, want := range map[string]string{"800": "800px", " 50% ": "50%", "12.5": "12.5px", "min(800px, 100%)": "min(800px, 100%)", "": ""} {
		if got := NormalizeCSSLength(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}

func TestCSSLengthProblemAcceptsAnyLengthAndRefusesWhatCouldEscapeTheStyle(t *testing.T) {
	for _, ok := range []string{"", "800", "50%", "600px", "40em", "min(800px, 100%)", "clamp(300px, 50%, 900px)", "calc(100% - 2rem)", "none"} {
		if problem := CSSLengthProblem(ok); problem != "" {
			t.Errorf("%q should be accepted, got %q", ok, problem)
		}
	}
	for _, bad := range []string{`50%"><script>`, "50%; color: red", "url(x)", "50% }", "min(800px", "a'b", "expression(1)"} {
		if problem := CSSLengthProblem(bad); problem == "" {
			t.Errorf("%q should be refused", bad)
		}
	}
}

func TestCSSLengthReadsTextOrANumberOfPixels(t *testing.T) {
	var wire struct {
		Width CSSLength `json:"w"`
	}
	for in, want := range map[string]string{`{"w":"50%"}`: "50%", `{"w":800}`: "800px", `{"w":"640"}`: "640"} {
		wire.Width = ""
		if err := json.Unmarshal([]byte(in), &wire); err != nil || string(wire.Width) != want {
			t.Errorf("%s: got %q, %v; want %q", in, wire.Width, err, want)
		}
	}
	if err := json.Unmarshal([]byte(`{"w":true}`), &wire); err == nil {
		t.Error("a boolean is not a length")
	}
}
