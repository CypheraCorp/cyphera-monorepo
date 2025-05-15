package bridge

import (
	"cyphera-api/internal/client/offramp"
)

// List of supported blockchain networks by Bridge.
// ChainID can be a common string identifier (e.g., "ethereum", "polygon") or a numeric ID if Bridge API uses that elsewhere.
// For now, using string identifiers based on the documentation table.
var supportedBridgeChainIDs = []offramp.NetworkData{
	{ChainID: "1", Network: "Ethereum"},      // Assuming ChainID "1" for Ethereum
	{ChainID: "137", Network: "Polygon"},     // Assuming ChainID "137" for Polygon
	{ChainID: "8453", Network: "Base"},       // Assuming ChainID "8453" for Base
	{ChainID: "42161", Network: "Arbitrum"},  // Assuming ChainID "42161" for Arbitrum
	{ChainID: "43114", Network: "Avalanche"}, // Assuming ChainID "43114" for Avalanche C-Chain
	{ChainID: "10", Network: "Optimism"},     // Assuming ChainID "10" for Optimism
}

// List of supported cryptocurrencies by Bridge.
var supportedBridgeCryptoCurrencies = []string{
	"USDC",
	"USDT",
	"DAI",
	"USDP",
	"PYUSD",
	"EURC",
}

// List of supported fiat currencies by Bridge.
var supportedBridgeFiatCurrencies = []string{
	"USD",
	"EUR",
	"MXN",
}

// List of generally supported jurisdictions by Bridge.
// Note: "US (excluding NY, FL, AK, LA)" and "International (excluding OFAC)" from docs needs careful handling.
// For a programmable list, we might list top-level supported countries and handle exclusions in logic or documentation.
// This list is a placeholder and should be refined based on precise operational capabilities and how this data is used.
var supportedBridgeJurisdictions = []string{
	"US", // For USD payouts (Note: Specific state exclusions like NY, FL, AK, LA apply as per Bridge docs).
	"MX", // For MXN payouts via SPEI/CLABE.

	// EEA+ Countries for EUR payouts via SEPA (based on provided list):
	"ALA", // Åland Islands
	"AUT", // Austria
	"BEL", // Belgium
	"BGR", // Bulgaria
	"HRV", // Croatia
	"CYP", // Cyprus
	"CZE", // Czechia
	"DNK", // Denmark
	"EST", // Estonia
	"FIN", // Finland
	"FRA", // France
	"GUF", // French Guiana
	"DEU", // Germany
	"GRC", // Greece
	"GLP", // Guadeloupe
	"HUN", // Hungary
	"ISL", // Iceland
	"IRL", // Ireland
	"ITA", // Italy
	"LVA", // Latvia
	"LIE", // Liechtenstein
	"LTU", // Lithuania
	"LUX", // Luxembourg
	"MLT", // Malta
	"MTQ", // Martinique
	"MYT", // Mayotte
	"NLD", // Netherlands
	"NOR", // Norway
	"POL", // Poland
	"PRT", // Portugal
	"REU", // Réunion
	"ROU", // Romania
	"MAF", // Saint Martin (French part)
	"SVK", // Slovakia
	"SVN", // Slovenia
	"ESP", // Spain
	"SWE", // Sweden
	"CHE", // Switzerland
	"GBR", // United Kingdom of Great Britain and Northern Ireland
	// Note: The documentation also states "most countries not on the OFAC sanctions list" for customer *onboarding*.
	// This list specifically targets jurisdictions for *fiat payouts* based on supported rails.
}

// Default KYC Redirect URI for Bridge.
const bridgeDefaultKYCRedirectURI = "https://cypherapay.com"

// Description of Bridge's fee structure.
const bridgeFeeStructureDetails = "Bridge passes through payment-method transaction fees at cost. USD payments: ACH $0.50, Same Day ACH $1, Wire $20. EUR payments: SEPA $1. Crypto withdrawals: Gas fees vary. Developers can configure additional fees. Refer to full Bridge documentation for complete details."

// Specific network names based on Bridge documentation for cross-referencing with the crypto table.
const (
	networkEthereum  = "Ethereum"
	networkPolygon   = "Polygon"
	networkBase      = "Base"
	networkArbitrum  = "Arbitrum"
	networkAvalanche = "Avalanche"
	networkOptimism  = "Optimism"
)

// Helper structure to map crypto to their supported networks based on Bridge documentation table.
// This helps in constructing the ProviderCapabilities more accurately.
type cryptoNetworkSupport struct {
	Crypto   string
	Networks []string
}

var bridgeCryptoNetworkSupport = []cryptoNetworkSupport{
	{Crypto: "USDC", Networks: []string{networkEthereum, networkPolygon, networkBase, networkArbitrum, networkAvalanche, networkOptimism}},
	{Crypto: "USDT", Networks: []string{networkEthereum}},
	{Crypto: "DAI", Networks: []string{networkEthereum}},
	{Crypto: "USDP", Networks: []string{networkEthereum}},
	{Crypto: "PYUSD", Networks: []string{networkEthereum, networkPolygon}},
	{Crypto: "EURC", Networks: []string{networkEthereum}},
}
