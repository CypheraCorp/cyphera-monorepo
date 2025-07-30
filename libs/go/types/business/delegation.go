package business

// DelegationStruct represents the delegation data structure
// This structure matches the format used by MetaMask delegation toolkit and the working integration test
type DelegationStruct struct {
	Delegate  string          `json:"delegate"`
	Delegator string          `json:"delegator"`
	Authority AuthorityStruct `json:"authority"`
	Caveats   []CaveatStruct  `json:"caveats"`
	Salt      string          `json:"salt"`
	Signature string          `json:"signature"`
}

// AuthorityStruct represents the authority information in a delegation
// Based on the working integration test format
type AuthorityStruct struct {
	Scheme    string `json:"scheme"`
	Signature string `json:"signature"`
	Signer    string `json:"signer"`
}

// CaveatStruct represents a single caveat in a delegation
// Based on MetaMask delegation toolkit: https://docs.metamask.io/delegation-toolkit/concepts/caveat-enforcers/
type CaveatStruct struct {
	Enforcer string `json:"enforcer"` // Address of the caveat enforcer contract
	Terms    string `json:"terms"`    // Encoded parameters defining the specific restrictions (hex string)
}
