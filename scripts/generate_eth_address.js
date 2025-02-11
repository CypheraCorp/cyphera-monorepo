const { ethers } = require('ethers');
const { SiweMessage } = require('siwe');

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

function generateNewWallet() {
    return ethers.Wallet.createRandom();
}

async function registerUser(address, message, signature, nonce) {
    try {
        const response = await fetch('https://api.billing.acta.link/api/ct/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'x-api-key': process.env.ACTALINK_API_KEY || '<your-api-key>',
            },
            body: JSON.stringify({
                address,
                message,
                signature,
                nonce,
            }),
            credentials: 'include',
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}, ${await response.text()}`);
        }

        return await response.json();
    } catch (error) {
        
        console.error('\n❌ Registration failed:', error.message);
        throw error;
    }
}

async function main() {
    const nonce = process.argv[2];
    
    if (!nonce) {
        console.error('\n❌ Error: Please provide a nonce as an argument');
        process.exit(1);
    }

    try {
        // // Generate new wallet
        // console.log('\n🔑 Generating new wallet...');
        // const newWallet = generateNewWallet();
        // console.log('   ├── Address:', newWallet.address);
        // console.log('   └── Private Key:', newWallet.privateKey);

        // Generate SIWE message and signature
        console.log('\n📝 Generating SIWE message and signature...');
        const { address, messageToSign, signature } = await generateAndSignMessage("0x97e200811ca21531a049c21d1f6c99daebf96fa839a53af2bae19629c45c92a9", nonce);
        console.log('   ├── Message:', messageToSign);
        console.log('   └── Signature:', signature);

        // Register user
        // console.log('\n🔄 Registering user...');
        // const registrationResult = await registerUser(address, messageToSign, signature, nonce);
        // console.log('\n✅ Registration successful!');
        // console.log('   └── Result:', JSON.stringify(registrationResult, null, 2));

    } catch (error) {
        console.error('\n❌ Error:', error.message);
        process.exit(1);
    }
}

main().catch(console.error); 