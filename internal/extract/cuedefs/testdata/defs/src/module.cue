package defs

// #Module: the unit an author publishes (0010:D4). It holds components
// and the values they read.
//
// A module names its components in #components; each is a #Component.
//
// WHY: maintainers only; never reaches a page.
//
// - one list item names #Widget
// - another item
//
//   an indented line
#Module: {
	apiVersion: "example.com/v1"
	kind:       "Module"

	metadata: {
		// The module's name, unique in its registry. See SPEC.md § 2.1.
		name!: #NameType

		// WHY name: a rationale block above a field.
		version!: string

		labels?: #LabelsType
	}

	// The components, keyed by id (0010:D9, 0010:D10).
	#components: #ComponentMap

	// Hidden machinery, dropped from the spec block.
	_count: len(#components)

	// A comment separated from the field by a blank line is dropped.

	debug: bool | *false // a trailing comment (0011:D2)

	if debug {
		trace!: string
	}

	_ok: true
	_ok: _count < 10
}

// #ComponentMap maps component ids to components.
#ComponentMap: [Id=string]: #Component

////////////////////////////////////////////////////////////////
// A banner group, dropped.

// Component is one deployable part of a #Module. Example: "web" "db"
#Component: {
	kind: "Component"
	#Widget

	spec: {...}
}

// #Widget: shared fields every component embeds.
#Widget: {
	// The widget's size.
	size: int & >=0
	...
}
