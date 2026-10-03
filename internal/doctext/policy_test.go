package doctext

import "testing"

func TestPolicy(t *testing.T) {
	for _, c := range []struct{ in, strip, link string }{
		{"Refused at admission (0010:D28).", "Refused at admission.",
			"Refused at admission ([0010:D28](/enhancements/0010/decisions/))."},
		{"Refused per 0010:D28:R2.", "Refused.", "Refused per [0010:D28:R2](/enhancements/0010/decisions/)."},
		{"See 0010:D28/D29 for both.", "See for both.", "See [0010:D28/D29](/enhancements/0010/decisions/) for both."},
		{"Both (0010:D28, 0011:D4).", "Both.",
			"Both ([0010:D28](/enhancements/0010/decisions/), [0011:D4](/enhancements/0011/decisions/))."},
		{"Open (0010:OQ3).", "Open.", "Open."},
		{"Mixed 0010:D28/OQ2 here.", "Mixed here.", "Mixed here."},
		{"Spec (SPEC.md § 2.2) and 0010 experiment 3.", "Spec and.", "Spec and."},
		{"In code `x // 0010:D28` stays code.", "In code `x // ` stays code.", "In code `x // ` stays code."},
		{"No citation here.", "No citation here.", "No citation here."},
	} {
		if got := Strip.Clean(c.in); got != c.strip {
			t.Errorf("strip %q = %q, want %q", c.in, got, c.strip)
		}
		if got := Link.Clean(c.in); got != c.link {
			t.Errorf("link %q = %q, want %q", c.in, got, c.link)
		}
	}
}

// A spec block's comment is cleaned by CleanCode, which strips under
// either policy: a link cannot live in a comment.
func TestCodeCommentsAlwaysStrip(t *testing.T) {
	if got := CleanCode("x: int // see 0010:D28"); got != "x: int // see" {
		t.Fatalf("%q", got)
	}
}
