/**
 * Ethereum Wallet Authentication Script
 * ===================================
 * 
 * This script handles authentication flow for the Acta billing system using an existing Ethereum wallet.
 * It implements Sign-In with Ethereum (SIWE) for secure authentication.
 * 
 * Key Features:
 * ------------
 * 1. Wallet Loading: Uses provided private key to load an Ethereum wallet
 * 2. Nonce Generation: Fetches a secure nonce from the server for SIWE
 * 3. SIWE Message Creation: Generates a structured message for Ethereum-based authentication
 * 4. User Verification: Checks if a wallet address is already registered
 * 5. Authentication Flow: Handles both registration and login based on user existence
 * 
 * Process Flow:
 * ------------
 * 1. Loads wallet from provided private key
 * 2. Fetches a secure nonce from the server
 * 3. Creates and signs a SIWE message using the wallet and nonce
 * 4. Checks if the wallet address is already registered
 * 5. Performs either registration or login based on user existence
 * 6. Returns authentication cookies and response data
 * 
 * API Endpoints Used:
 * ----------------
 * - GET  /api/ct/nonce           - Retrieves a secure nonce
 * - GET  /api/ct/isuseravailable - Checks if a user exists
 * - POST /api/ct/register        - Registers a new user
 * - POST /api/ct/login          - Authenticates existing user
 * 
 * Usage:
 * -----
 * node generate_eth_address.js <private_key>
 * 
 * Arguments:
 * ---------
 * private_key: Ethereum wallet private key (required, 64 hex characters without 0x prefix)
 * 
 * Environment Variables:
 * -------------------
 * ACTALINK_API_KEY - API key for authentication (required)
 * 
 * Dependencies:
 * -----------
 * - ethers: Ethereum wallet and signing functionality
 * - siwe: Sign-In with Ethereum message standard
 */

const { ethers } = require('ethers');
const { SiweMessage } = require('siwe');

async function getNonce() {
    try {
        const response = await fetch('https://api.billing.acta.link/api/ct/nonce', {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
                'x-api-key': process.env.ACTALINK_API_KEY || '<your-api-key>',
            },
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}, ${await response.text()}`);
        }

        const data = await response.json();
        return data;
    } catch (error) {
        console.error('\n❌ Nonce generation failed:', error.message);
        throw error;
    }
}

async function generateAndSignMessage(privateKeyHex, nonce) {
    const wallet = new ethers.Wallet(privateKeyHex);
    const address = wallet.address;
    
    // Create SIWE message
    const siweMessage = new SiweMessage({
        domain: 'billing.acta.link',
        address: address,
        statement: 'Sign in with Ethereum to Acta link',
        uri: 'https://billing.acta.link',
        version: '1',
        chainId: 1,
        nonce: nonce
    });
    
    const messageToSign = siweMessage.prepareMessage();
    const signature = await wallet.signMessage(messageToSign);
    
    return { address, messageToSign, signature };
}

async function loadWallet(privateKey) {
    try {
        if (!privateKey) {
            throw new Error('Private key is required');
        }
        // Add 0x prefix if not present
        const formattedKey = privateKey.startsWith('0x') ? privateKey : `0x${privateKey}`;
        return new ethers.Wallet(formattedKey);
    } catch (error) {
        throw new Error(`Invalid private key: ${error.message}`);
    }
}

async function loginRegisterUser(address, message, signature, nonce, type) {
    try {
        const body = JSON.stringify({
            address,
            message,
            signature,
            nonce,
        });
        console.log(body);
        const response = await fetch(`https://api.billing.acta.link/api/ct/${type}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'x-api-key': process.env.ACTALINK_API_KEY || '<your-api-key>',
            },
            body: body,
            credentials: 'include',
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}, ${await response.text()}`);
        }

        const data = await response.json();
        const cookies = response.headers.get('set-cookie');
        return { data, cookies };
    } catch (error) {
        console.error('\n❌ Registration failed:', error.message);
        throw error;
    }
}

async function checkUserExists(address) {
    try {
        const response = await fetch(`https://api.billing.acta.link/api/ct/isuseravailable?address=${address}`, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
                'x-api-key': process.env.ACTALINK_API_KEY || '<your-api-key>',
            },
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}, ${await response.text()}`);
        }

        const data = await response.json();
        return data.message === "exists";
    } catch (error) {
        console.error('\n❌ User check failed:', error.message);
        throw error;
    }
}

async function main() {
    const privateKey = process.argv[2];
    
    if (!privateKey) {
        console.error('\n❌ Error: Please provide a private key as an argument');
        console.error('Usage: node generate_eth_address.js <private_key>');
        process.exit(1);
    }

    try {
        // Load wallet from private key
        console.log('\n🔑 Loading wallet...');
        const wallet = await loadWallet(privateKey);
        console.log('   └── Address:', wallet.address);

        // Get nonce from server
        console.log('\n🔄 Fetching nonce from server...');
        const nonce = await getNonce();
        console.log('   └── Nonce:', nonce);

        // Generate SIWE message and signature
        console.log('\n📝 Generating SIWE message and signature...');
        const { address, messageToSign, signature } = await generateAndSignMessage(wallet.privateKey, nonce);
        console.log('   ├── Message:', messageToSign);
        console.log('   └── Signature:', signature);

        // Check if user exists
        console.log('\n🔍 Checking if user exists...');
        const userExists = await checkUserExists(address);
        
        const type = userExists ? 'login' : 'register';
        console.log(`\n${userExists ? '👤 User exists, proceeding with login' : '📝 User does not exist, proceeding with registration'}...`);
        
        const result = await loginRegisterUser(address, messageToSign, signature, nonce, type);
        console.log(`\n✅ ${userExists ? 'Login' : 'Registration'} successful!`);
        console.log('   ├── Cookie:', result.cookies);
        console.log('   └── Result:', JSON.stringify(result.data, null, 2));

    } catch (error) {
        console.error('\n❌ Error:', error.message);
        process.exit(1);
    }
}

main().catch(console.error); 