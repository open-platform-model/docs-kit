// The demo catalog: a small self-contained catalog for the extractor's
// tests.
package demo

import (
	res "example.com/catalogs/demo/resources/v1"
	tra "example.com/catalogs/demo/traits/v1alpha1"
	tr "example.com/catalogs/demo/traits/v1beta1"
	bp "example.com/catalogs/demo/blueprints/v1"
	t "example.com/catalogs/demo/transformers"
)

metadata: {
	modulePath: "example.com/catalogs/demo@v1"
	version:    "1.2.3"
}

#resources: {
	(res.#ContainerResource.metadata.fqn): res.#ContainerResource
	(res.#QueueResource.metadata.fqn):     res.#QueueResource
}

#traits: {
	(tra.#BackupTrait.metadata.fqn): tra.#BackupTrait
	(tr.#BackupTrait.metadata.fqn):  tr.#BackupTrait
	(tr.#ScalingTrait.metadata.fqn): tr.#ScalingTrait
}

#blueprints: {
	(bp.#WebBlueprint.metadata.fqn): bp.#WebBlueprint
}

#transformers: {
	deployment: t.#DeploymentTransformer
	service:    t.#ServiceTransformer
}
