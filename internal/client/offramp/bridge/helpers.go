package bridge

import offramp "cyphera-api/internal/client/offramp"

// This function generates the final list of supportedChainIDs and supportedCryptoCurrencies
// based on the detailed cryptoNetworkSupport table, ensuring consistency.
func getConsolidatedBridgeCapabilities() (chainIDs []offramp.NetworkData, cryptoCurrencies []string) {
	uniqueChainIDs := make(map[string]offramp.NetworkData)
	uniqueCryptoCurrencies := make(map[string]bool)

	nameToID := make(map[string]string)
	for _, nd := range supportedBridgeChainIDs { // Use the predefined list for ChainID mapping
		nameToID[nd.Network] = nd.ChainID
	}

	for _, support := range bridgeCryptoNetworkSupport {
		uniqueCryptoCurrencies[support.Crypto] = true
		for _, networkName := range support.Networks {
			chainID, ok := nameToID[networkName]
			if !ok {
				// Fallback or error if a network name in support table isn't in main chain ID list
				// For now, using network name as ID if not found, but this should be consistent.
				chainID = networkName
			}
			uniqueChainIDs[networkName+"_"+chainID] = offramp.NetworkData{ChainID: chainID, Network: networkName}
		}
	}

	for _, data := range uniqueChainIDs {
		chainIDs = append(chainIDs, data)
	}
	for crypto := range uniqueCryptoCurrencies {
		cryptoCurrencies = append(cryptoCurrencies, crypto)
	}
	return
}
