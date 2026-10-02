package transformers

import (
	res "example.com/catalogs/demo/resources/v1"
	tr "example.com/catalogs/demo/traits/v1beta1"
)

#DeploymentTransformer: {
	metadata: {
		name:        "deployment"
		fqn:         "example.com/catalogs/demo/transformers/deployment@v1"
		description: "Renders a stateless workload as a Deployment | with <pods>"
	}
	requiredLabels: "workload-type":                          "stateless"
	requiredResources: (res.#ContainerResource.metadata.fqn): res.#ContainerResource
	optionalTraits: (tr.#BackupTrait.metadata.fqn):           tr.#BackupTrait
}

#ServiceTransformer: {
	metadata: {
		name:        "service"
		fqn:         "example.com/catalogs/demo/transformers/service@v1"
		description: "Exposes a container as a Service"
	}
	optionalResources: (res.#ContainerResource.metadata.fqn): res.#ContainerResource
}
