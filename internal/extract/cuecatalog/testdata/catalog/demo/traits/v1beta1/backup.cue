package v1beta1

import res "example.com/catalogs/demo/resources/v1"

// Scheduled backup policy for a component's state. The deployment
// transformer reads it when present.
#BackupTrait: {
	kind: "Trait"
	metadata: {
		modulePath:     "example.com/catalogs/demo/traits/v1beta1"
		name:           "backup"
		apiVersion:     "v1beta1"
		catalogVersion: "1.2.3"
		fqn:            "example.com/catalogs/demo/traits/backup@v1beta1"
		description:    "Scheduled backup policy for a component's state"
	}
	fulfilment: *"catalog" | "provider"
	optional:   bool | *true
	appliesTo: [res.#ContainerResource]
	spec: backup: {
		schedule!: string
		// Copies kept, under the keep-more rule.
		keep: int & >0 | *7
	}
}
