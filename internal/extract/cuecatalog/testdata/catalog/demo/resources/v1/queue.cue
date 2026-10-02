package v1

// A message queue a component declares. Nothing in this catalog renders it.
#QueueResource: {
	kind: "Resource"
	metadata: {
		modulePath:     "example.com/catalogs/demo/resources/v1"
		name:           "queue"
		apiVersion:     "v1"
		catalogVersion: "1.2.3"
		fqn:            "example.com/catalogs/demo/resources/queue@v1"
		description:    "A message queue a component declares"
	}
	fulfilment: *"catalog" | "provider"
	spec: queue: {
		// Messages kept at most.
		depth: int & >0 | *100
	}
}
