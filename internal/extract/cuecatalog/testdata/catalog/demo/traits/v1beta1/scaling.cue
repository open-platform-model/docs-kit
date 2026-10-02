package v1beta1

import res "example.com/catalogs/demo/resources/v1"

// Replica bounds for a component. Advisory: see docs/scaling-notes.md, and
// a pattern like {{< x >}} stays escaped.
#ScalingTrait: {
	kind: "Trait"
	metadata: {
		modulePath:     "example.com/catalogs/demo/traits/v1beta1"
		name:           "scaling"
		apiVersion:     "v1beta1"
		catalogVersion: "1.2.3"
		fqn:            "example.com/catalogs/demo/traits/scaling@v1beta1"
		description:    "Replica bounds for a component"
	}
	fulfilment: *"catalog" | "provider"
	optional:   bool | *true
	appliesTo: [res.#ContainerResource]
	spec: scaling: {
		min: int | *1
		max: int | *3
		// Per-zone overrides, by zone name.
		zones?: [string]: int
	}
}
