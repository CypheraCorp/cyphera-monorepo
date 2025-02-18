/**
 * Ethereum Wallet Generator Script
 * ==============================
 * 
 * This script generates a new Ethereum wallet and outputs its private key.
 * Use this for development and testing purposes only.
 * Never share or commit private keys.
 * 
 * Usage:
 * -----
 * node create_private_key.js
 * 
 * Output:
 * -------
 * - Wallet Address
 * - Private Key (with and without 0x prefix)
 * 
 * Dependencies:
 * -----------
 * - ethers: Ethereum wallet functionality
 */

const { ethers } = require('ethers');

function generateWallet() {
    // Create a new random wallet
    const wallet = ethers.Wallet.createRandom();
    
    console.log('\n🔑 New Ethereum Wallet Generated');
    console.log('   ├── Address:', wallet.address);
    console.log('   ├── Private Key (with 0x):', wallet.privateKey);
    console.log('   └── Private Key (without 0x):', wallet.privateKey.slice(2));
    
    // Add warning message
    console.log('\n⚠️  WARNING: Keep this private key secure and never share it!');
    console.log('   This key provides full control over the associated wallet.\n');
}

// Execute
generateWallet();
