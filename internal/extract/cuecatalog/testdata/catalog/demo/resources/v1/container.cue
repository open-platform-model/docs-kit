package v1

import k8s "example.com/catalogs/demo/schemas/kubernetes/core/v1"

// One container image a component runs. The deployment transformer renders
// it (0010:D28).
#ContainerResource: {
	kind: "Resource"
	metadata: {
		modulePath:     "example.com/catalogs/demo/resources/v1"
		name:           "container"
		apiVersion:     "v1"
		catalogVersion: "1.2.3"
		fqn:            "example.com/catalogs/demo/resources/container@v1"
		description:    "One container image a component runs"
		labels: "resource.opmodel.dev/category": "workload"
	}
	fulfilment: *"catalog" | "provider"
	spec: container: #ContainerSchema
}

// WHY a separate schema: the blueprint composes it (0015 D1).

// The container's settings.
#ContainerSchema: {
	// The container name, unique in the component.
	name!: string
	// How many replicas run (0010:D9).
	replicas: int | *1
	image!:   string
	// Environment variables, the vendored Kubernetes shape.
	env?: [...k8s.#EnvVar]
}
