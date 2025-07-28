package business

// DelegationStruct represents the delegation data structure
type DelegationStruct struct {
	Delegate  string         `json:"delegate"`
	Delegator string         `json:"delegator"`
	Authority string         `json:"authority"`
	Caveats   []CaveatStruct `json:"caveats"`
	Salt      string         `json:"salt"`
	Signature string         `json:"signature"`
}

// CaveatStruct represents a single caveat in a delegation
type CaveatStruct struct {
	// TODO: add caveat fields
	// Define the fields for CaveatStruct based on your needs
}
