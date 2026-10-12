package doctext

import (
	"strings"
	"testing"
)

// The old citation forms (NNNN:Dn, NNNN Dn, NNNN:OQn) stay in these inputs
// on purpose: a docs revision builds an old release tag whose comments still
// hold them, and both policies remove them.
func TestPolicy(t *testing.T) {
	for _, c := range []struct{ in, strip, link string }{
		{"Refused at admission (0010:D28).", "Refused at admission.", "Refused at admission."},
		{"Refused per 0010:D28:R2.", "Refused.", "Refused."},
		{"See 0010:D28/D29 for both.", "See for both.", "See for both."},
		{"Both (0010:D28, 0011:D4).", "Both.", "Both."},
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
		{"citation form", "Per enhancement 0010:D28, refused.", "Per, refused.", "Per, refused."},
		{"citation with a space", "Per enhancement 0010 D28, refused.", "Per, refused.", "Per, refused."},
		{"open question", "Open (enhancement 0010:OQ3).", "Open.", "Open."},
		{"experiment pointer", "Measured in enhancement 0010 experiment 3.", "Measured in.", "Measured in."},
		{"mixed", "Enhancement 0013 (0010:D4) and enhancement 0014 experiment 2.", "Enhancement 0013 and.",
			"[Enhancement 0013](/enhancements/0013/) and."},
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

// Under strip, a parenthesis that holds only an enhancement reference goes,
// with the space before it. Under link it stays and its reference is linked.
func TestLoneEnhancementCitation(t *testing.T) {
	for _, c := range []struct{ name, in, strip, link string }{
		{"end of sentence", "It refuses the render (enhancement 0013).", "It refuses the render.",
			"It refuses the render ([enhancement 0013](/enhancements/0013/))."},
		{"middle of sentence", "The rule (enhancement 0013) holds here.", "The rule holds here.",
			"The rule ([enhancement 0013](/enhancements/0013/)) holds here."},
		{"start of a line", "(enhancement 0013) The rule holds.", "The rule holds.",
			"([enhancement 0013](/enhancements/0013/)) The rule holds."},
		{"two in one sentence", "A (enhancement 0013) and B (enhancement 0014).", "A and B.",
			"A ([enhancement 0013](/enhancements/0013/)) and B ([enhancement 0014](/enhancements/0014/))."},
		{"capital", "The rule (Enhancement 0013).", "The rule.", "The rule ([Enhancement 0013](/enhancements/0013/))."},
		{"plural list", "The rule (enhancements 0013 and 0019).", "The rule.",
			"The rule (enhancements [0013](/enhancements/0013/) and [0019](/enhancements/0019/))."},
		{"plural comma list", "The rule (enhancements 0013, 0014 and 0019), then more.", "The rule, then more.",
			"The rule (enhancements [0013](/enhancements/0013/), [0014](/enhancements/0014/) and [0019](/enhancements/0019/)), then more."},
		{"space inside the parenthesis", "The rule ( enhancement 0013 ).", "The rule.",
			"The rule ( [enhancement 0013](/enhancements/0013/) )."},
		{"code span stays", "Write `(enhancement 0013)` here.", "Write `(enhancement 0013)` here.", "Write `(enhancement 0013)` here."},
		{"double backtick span", "Use ``a`b`` and `(enhancement 0013)` here (enhancement 0014).", "Use ``a`b`` and `(enhancement 0013)` here.",
			"Use ``a`b`` and `(enhancement 0013)` here ([enhancement 0014](/enhancements/0014/))."},
		{"unclosed backtick", "A ` mark (enhancement 0013).", "A ` mark.", "A ` mark ([enhancement 0013](/enhancements/0013/))."},
		{"code span stays, prose goes", "Write `(enhancement 0013)` here (enhancement 0014).", "Write `(enhancement 0013)` here.",
			"Write `(enhancement 0013)` here ([enhancement 0014](/enhancements/0014/))."},
		{"other words stay", "The design (see enhancement 0013 for it) is final.", "The design (see enhancement 0013 for it) is final.",
			"The design (see [enhancement 0013](/enhancements/0013/) for it) is final."},
		{"words before the reference stay", "The rule (per enhancement 0013).", "The rule (per enhancement 0013).",
			"The rule (per [enhancement 0013](/enhancements/0013/))."},
		{"other parenthesis stays", "The year (2026) and a note (see docs).", "The year (2026) and a note (see docs).",
			"The year (2026) and a note (see docs)."},
		{"fewer than four digits", "The rule (enhancement 13).", "The rule (enhancement 13).", "The rule (enhancement 13)."},
		{"five digits", "The rule (enhancement 00131).", "The rule (enhancement 00131).", "The rule (enhancement 00131)."},
		{"markdown link destination", "See [the entry](enhancement 0013) now.", "See [the entry](enhancement 0013) now.",
			"See [the entry]([enhancement 0013](/enhancements/0013/)) now."},
		{"no lone citation", "Added by enhancement 0013 in September.", "Added by enhancement 0013 in September.",
			"Added by [enhancement 0013](/enhancements/0013/) in September."},
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

// A spec block's comment is cleaned by CleanCode, which leaves a lone
// enhancement parenthesis as written: only prose loses it.
func TestCodeBlockKeepsLoneEnhancement(t *testing.T) {
	in := "x: int // the rule (enhancement 0013)"
	if got := CleanCode(in); got != in {
		t.Fatalf("%q", got)
	}
}

func TestStripCleanParagraphs(t *testing.T) {
	got := Strip.CleanParagraphs([]string{"One (enhancement 0013).", "(enhancement 0014)", "Two."})
	want := []string{"One.", "Two."}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q, want %q", got, want)
	}
}
