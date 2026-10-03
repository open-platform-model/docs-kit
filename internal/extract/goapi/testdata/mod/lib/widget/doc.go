// Package widget makes widgets (0010:D28). It is the fixture of the go-api
// extractor.
//
// A [Widget] comes from [New] and spins with [Widget.Spin]; a
// [gadget.Gadget] holds one. Every call takes a [context.Context].
// Values are `cue` values, and <b>raw HTML</b> and {{< figure >}} stay text.
// WHY: this line is for maintainers.
//
// # Spinning
//
// Spin a widget:
//
//	w := widget.New()
//	w.Spin() // `once`
//
// The rules:
//   - a widget spins once;
//   - 1. is not a list here.
//
// See [the docs] for more.
//
// [the docs]: https://example.com/docs?a=(b)
package widget
