package defs

import "strings"

// NameType: an RFC 1123 DNS label. Usage: "web" => valid
#NameType: string & =~"^[a-z0-9]([-a-z0-9]*[a-z0-9])?$" & strings.MinRunes(1) & strings.MaxRunes(63)

// #LabelsType is a map of label keys to values.
#LabelsType: [string]: string

// #Mode is either fast or safe.
#Mode: "fast" | "safe"

// #Unused is declared but not on any page.
#Unused: string
