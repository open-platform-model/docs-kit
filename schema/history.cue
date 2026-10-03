package schema

#History: {
	schema:  "docs.opmodel.dev/history/v1"
	project: #Project
	tool:    #SemVer // the opm-docs that computed it
	floor:   =~"^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$"
	segments: [#Segment, #Segment, ...#Segment] // ordered: minors ascending, edge last
	compared: [...{from: #Segment, to: #Segment, mode: "full" | "paths"}]
	members: [string]:                           #MemberHistory // keyed by FQN
	removed: [#Segment]: [#Removed, ...#Removed] // a segment key only when something was removed in it
	lineage: [string]: [#Segment]: [string, ...string] // "<kind>/<name>": segment: apiVersions, newest first (C8 order)
}

#MemberHistory: {
	kind:         "resource" | "trait" | "blueprint"
	name:         string
	apiVersion:   string
	first:        #Segment // the first segment that has it
	firstIsFloor: bool     // first == floor: the site writes "in <floor> or earlier", never "added in"
	in: [#Segment, ...#Segment]
	changes: [#Segment]: [#Change, ...#Change] // a segment key only when the member changed against its previous segment
}

#Change: {
	op:   "added" | "removed" | "presence" | "type" | "default" | "ref" | "spec"
	path: string // a spec.fields path; "" for op "spec"
	from: string | null
	to:   string | null
}

#Removed: {
	fqn:        string
	kind:       "resource" | "trait" | "blueprint"
	name:       string
	apiVersion: string
	lastIn:     #Segment // the last segment that had it: the site links <root><lastIn>/<page>/
	page:       string   // its page in lastIn, "traits/backup-v1alpha1"
}
