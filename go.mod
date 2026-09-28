module satellion.com/passmcp-graph

go 1.26.8

require (
	go.yaml.in/yaml/v3 v3.0.5
	satellion.com/passmcp-reporting v0.0.1
)

// Until satellion.com/passmcp-reporting v0.0.1 is tagged, it comes from the
// sibling checkout. The release replaces this with the tagged version.
replace satellion.com/passmcp-reporting => ../passmcp-reporting
