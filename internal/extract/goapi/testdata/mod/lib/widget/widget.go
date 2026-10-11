package widget

import "context"

// Mode is how a widget spins.
type Mode int

// The modes.
const (
	// Slow spins slowly.
	Slow Mode = iota
	Fast      // spins fast
)

// Version is the widget version.
const Version = "1.0"

// Default is the default widget.
var Default = New()

// Widget is a thing that spins.
type Widget struct {
	// Name names the widget.
	Name string
	// WHY: maintainers only.
	Size   int // in mm (0010:D3)
	hidden bool
}

// New returns a [Widget].
func New() *Widget { return &Widget{} }

// Spin spins w; see [Widget.Stop] and [WidgetSpin].
func (w *Widget) Spin(ctx context.Context) error { return nil }

// Stop stops w.
func (w Widget) Stop() {}

// WidgetSpin collides with the anchor of [Widget.Spin].
func WidgetSpin() {}

func Undocumented() {}

func unexported() {}
