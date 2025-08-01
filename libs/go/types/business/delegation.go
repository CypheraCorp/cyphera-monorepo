package business

// DelegationStruct represents the delegation data structure
// This structure matches the format used by MetaMask delegation toolkit
// Reference: https://docs.metamask.io/delegation-toolkit/how-to/create-delegation/
type DelegationStruct struct {
	Delegate  string         `json:"delegate"`  // The address being delegated to
	Delegator string         `json:"delegator"` // The address creating the delegation
	Authority string         `json:"authority"` // Hex-encoded authority (typically the delegator)
	Caveats   []CaveatStruct `json:"caveats"`   // Restrictions on the delegation
	Salt      string         `json:"salt"`      // Random value for uniqueness
	Signature string         `json:"signature"` // The delegation signature
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
