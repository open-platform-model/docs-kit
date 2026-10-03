// A cue-definitions source over the fixture package.
bundles: defs: {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/definitions/"]}
	version: {from: "tag", prefix: "v"}
	sources: [{
		kind:    "cue-definitions"
		package: "./src"
		skip: ["*_pins.cue"]
		section:     "reference/definitions/"
		title:       "Definitions"
		description: "Every definition of the fixture."
		weight:      2
		pages: [
			{file: "modules", title: "Modules", description: "The module and its components.", definitions: ["#Module", "#Component", "#Widget"]},
			{file: "types", title: "Types", description: "Constraint types.", definitions: ["#NameType", "#LabelsType", "#Mode"]},
		]
		exclude: {
			"#ComponentMap": "map shorthand"
			"#Unused":       "not documented"
			"#MODE":         "differs from #Mode only in case"
		}
	}]
}
