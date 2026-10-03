---
title: "lib/widget"
description: "Package widget makes widgets."
type: reference
---

```go
import "example.com/widgets/lib/widget"
```

Package widget makes widgets. It is the fixture of the go-api extractor.

A [Widget](/docs/reference/go-api/widget/#widget) comes from [New](/docs/reference/go-api/widget/#new) and spins with [Widget.Spin](/docs/reference/go-api/widget/#widgetspin-1); a [gadget.Gadget](/docs/reference/go-api/gadget/#gadget) holds one. Every call takes a [context.Context](https://pkg.go.dev/context#Context). Values are `cue` values, and \<b\>raw HTML\</b\> and \{\{\< figure \>\}\} stay text.

### Spinning

Spin a widget:

```text
w := widget.New()
w.Spin() // `once`
```

The rules:

- a widget spins once;
- 1\. is not a list here.

See [the docs](https://example.com/docs?a=%28b%29) for more.

## Constants

### Version

```go
const Version = "1.0"
```

Version is the widget version.

## Variables

### Default

```go
var Default = New()
```

Default is the default widget.

## Functions

### Undocumented

```go
func Undocumented()
```

### WidgetSpin

```go
func WidgetSpin()
```

WidgetSpin collides with the anchor of [Widget.Spin](/docs/reference/go-api/widget/#widgetspin-1).

## Types

### Mode

```go
type Mode int
```

Mode is how a widget spins.

#### Slow

```go
const (
	// Slow spins slowly.
	Slow Mode = iota
	Fast      // spins fast
)
```

The modes.

### Widget

```go
type Widget struct {
	// Name names the widget.
	Name string

	Size int // in mm
	// contains filtered or unexported fields
}
```

Widget is a thing that spins.

#### New

```go
func New() *Widget
```

New returns a [Widget](/docs/reference/go-api/widget/#widget).

#### Widget.Spin

```go
func (w *Widget) Spin(ctx context.Context) error
```

Spin spins w; see [Widget.Stop](/docs/reference/go-api/widget/#widgetstop) and [WidgetSpin](/docs/reference/go-api/widget/#widgetspin).

#### Widget.Stop

```go
func (w Widget) Stop()
```

Stop stops w.
