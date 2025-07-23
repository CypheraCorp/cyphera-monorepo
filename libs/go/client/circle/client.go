package circle

import (
	"github.com/cyphera/cyphera-api/libs/go/client/http"
	"fmt"
)

const (
	CircleAPIBaseURL = "https://api.circle.com/v1/w3s"
)

// Blockchain constants for use with Circle API
const (
	// Mainnet blockchains
	BlockchainETH   = "ETH"
	BlockchainAVAX  = "AVAX"
	BlockchainMATIC = "MATIC"
	BlockchainSOL   = "SOL"
	BlockchainARB   = "ARB"
	BlockchainNEAR  = "NEAR"
	BlockchainEVM   = "EVM"
	BlockchainUNI   = "UNI"
	BlockchainOP    = "OP"
	BlockchainBASE  = "BASE"

	// Testnet blockchains
	BlockchainETHSepolia  = "ETH-SEPOLIA"
	BlockchainAVAXFuji    = "AVAX-FUJI"
	BlockchainMATICAmoy   = "MATIC-AMOY"
	BlockchainSOLDevnet   = "SOL-DEVNET"
	BlockchainARBSepolia  = "ARB-SEPOLIA"
	BlockchainNEARTestnet = "NEAR-TESTNET"
	BlockchainEVMTestnet  = "EVM-TESTNET"
	BlockchainUNISepolia  = "UNI-SEPOLIA"
	BlockchainOPSepolia   = "OP-SEPOLIA"
	BlockchainBASESepolia = "BASE-SEPOLIA"
)

// AllBlockchains is a slice containing all supported blockchain values
var AllBlockchains = []string{
	BlockchainETH, BlockchainAVAX, BlockchainMATIC, BlockchainSOL,
	BlockchainARB, BlockchainNEAR, BlockchainEVM, BlockchainUNI,
	BlockchainETHSepolia, BlockchainAVAXFuji, BlockchainMATICAmoy,
	BlockchainSOLDevnet, BlockchainARBSepolia, BlockchainNEARTestnet,
	BlockchainEVMTestnet, BlockchainUNISepolia, BlockchainOPSepolia,
	BlockchainBASESepolia,
}

// ValidateBlockchains checks if the provided blockchains are valid
// Returns an error if no blockchains are provided or if any blockchain is invalid
func ValidateBlockchains(blockchains []string) error {
	if len(blockchains) == 0 {
		return fmt.Errorf("at least one blockchain must be specified")
	}

	// Create a map for faster lookup of valid blockchains
	validBlockchains := make(map[string]bool)
	for _, chain := range AllBlockchains {
		validBlockchains[chain] = true
	}

	// Check each requested blockchain
	for _, chain := range blockchains {
		if !validBlockchains[chain] {
			return fmt.Errorf("invalid blockchain specified: %s", chain)
		}
	}

	return nil
}

type CircleClient struct {
	apiKey     string
	httpClient *http.HTTPClient
}

func NewCircleClient(apiKey string) *CircleClient {
	httpClient := http.NewHTTPClient(
		http.WithBaseURL(CircleAPIBaseURL),
	)
	return &CircleClient{
		httpClient: httpClient,
		apiKey:     apiKey,
	}
}
