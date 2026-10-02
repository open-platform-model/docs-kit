package v1alpha1

import res "example.com/catalogs/demo/resources/v1"

// WHY: provider-fulfilled, so this catalog ships no transformer (0010 D37).

// Scheduled backup policy for a component's state. A platform's backup
// adapter reads it.
//
// Exactly one provider serves it (0010:D32).
#BackupTrait: {
	kind: "Trait"
	metadata: {
		modulePath:     "example.com/catalogs/demo/traits/v1alpha1"
		name:           "backup"
		apiVersion:     "v1alpha1"
		catalogVersion: "1.2.3"
		fqn:            "example.com/catalogs/demo/traits/backup@v1alpha1"
		description:    "Scheduled backup policy for a component's state"
		labels: "trait.opmodel.dev/category": "storage"
	}
	fulfilment: "provider"
	optional:   bool | *false
	appliesTo: [res.#ContainerResource]
	spec: backup: #BackupSchema
}

#Backup: {
	#traits: (#BackupTrait.metadata.fqn): #BackupTrait
}

#BackupSchema: {
	schedule!: string
	keep?:     int & >0
}
