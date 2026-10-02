package v1

import res "example.com/catalogs/demo/resources/v1"

// A stateless web workload. One container, matched to the deployment
// transformer by its label.
#WebBlueprint: {
	kind: "Blueprint"
	metadata: {
		modulePath:     "example.com/catalogs/demo/blueprints/v1"
		name:           "web"
		apiVersion:     "v1"
		catalogVersion: "1.2.3"
		fqn:            "example.com/catalogs/demo/blueprints/web@v1"
		description:    "A stateless web workload"
	}
	composedResources: [res.#ContainerResource]
	composedTraits: []
	matchLabels: "workload-type"!: "stateless"
	spec: web:                     #WebSchema
}

#WebSchema: {
	container: res.#ContainerSchema
	// The port the web server listens on.
	port: int | *8080
}
