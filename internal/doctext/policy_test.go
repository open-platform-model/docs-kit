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

func TestEnhancementReferences(t *testing.T) {
	for _, c := range []struct{ name, in, strip, link string }{
		{"one", "Added by enhancement 0013 in September.", "Added by enhancement 0013 in September.",
			"Added by [enhancement 0013](/enhancements/0013/) in September."},
		{"capital", "Enhancement 0013 owns it.", "Enhancement 0013 owns it.",
			"[Enhancement 0013](/enhancements/0013/) owns it."},
		{"end of sentence", "See enhancement 0013.", "See enhancement 0013.", "See [enhancement 0013](/enhancements/0013/)."},
		{"colon after", "enhancement 0013: the plan.", "enhancement 0013: the plan.", "[enhancement 0013](/enhancements/0013/): the plan."},
		{"two", "Split between enhancements 0013 and 0014.", "Split between enhancements 0013 and 0014.",
			"Split between enhancements [0013](/enhancements/0013/) and [0014](/enhancements/0014/)."},
		{"three", "Enhancements 0013, 0014 and 0015 agree.", "Enhancements 0013, 0014 and 0015 agree.",
			"Enhancements [0013](/enhancements/0013/), [0014](/enhancements/0014/) and [0015](/enhancements/0015/) agree."},
		{"plural, one id", "Under enhancements 0013 only.", "Under enhancements 0013 only.",
			"Under enhancements [0013](/enhancements/0013/) only."},
		{"singular takes one id", "enhancement 0013 and 0014 differ.", "enhancement 0013 and 0014 differ.",
			"[enhancement 0013](/enhancements/0013/) and 0014 differ."},
		{"two references", "enhancement 0013 or enhancement 0014.", "enhancement 0013 or enhancement 0014.",
			"[enhancement 0013](/enhancements/0013/) or [enhancement 0014](/enhancements/0014/)."},
		{"code span", "Run `enhancement 0013` to see it.", "Run `enhancement 0013` to see it.", "Run `enhancement 0013` to see it."},
		{"five digits", "enhancement 00131 is not one.", "enhancement 00131 is not one.", "enhancement 00131 is not one."},
		{"no number", "an enhancement is a plan.", "an enhancement is a plan.", "an enhancement is a plan."},
		{"citation form", "Per enhancement 0010:D28, refused.", "Per, refused.",
			"Per enhancement [0010:D28](/enhancements/0010/decisions/), refused."},
		{"citation with a space", "Per enhancement 0010 D28, refused.", "Per, refused.", "Per, refused."},
		{"open question", "Open (enhancement 0010:OQ3).", "Open.", "Open."},
		{"experiment pointer", "Measured in enhancement 0010 experiment 3.", "Measured in.", "Measured in."},
		{"mixed", "Enhancement 0013 (0010:D4) and enhancement 0014 experiment 2.", "Enhancement 0013 and.",
			"[Enhancement 0013](/enhancements/0013/) ([0010:D4](/enhancements/0010/decisions/)) and."},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := Strip.Clean(c.in); got != c.strip {
				t.Errorf("strip %q = %q, want %q", c.in, got, c.strip)
			}
			if got := Link.Clean(c.in); got != c.link {
				t.Errorf("link %q = %q, want %q", c.in, got, c.link)
			}
		})
	}
}

// LinkEnhancements is what the crd renderer calls on prose the extractor
// kept plain.
func TestLinkEnhancements(t *testing.T) {
	in := "Owned by enhancements 0013 and 0014, not enhancement 0010 experiment 1."
	want := "Owned by enhancements [0013](/enhancements/0013/) and [0014](/enhancements/0014/), not enhancement 0010 experiment 1."
	if got := LinkEnhancements(in); got != want {
		t.Errorf("LinkEnhancements = %q, want %q", got, want)
	}
}
