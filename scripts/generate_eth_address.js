const { ethers } = require('ethers');
const { SiweMessage } = require('siwe');

async function generateAndSignMessage(privateKeyHex, nonce) {
    const wallet = new ethers.Wallet(privateKeyHex);
    const address = wallet.address;
    
    console.log('Private Key:', privateKeyHex);
    console.log('Address:', address);
    
    // Create SIWE message
    const siweMessage = new SiweMessage({
        domain: 'billing.acta.link',
        address: address,
        statement: 'Sign in with Ethereum to Acta.',
        uri: 'https://billing.acta.link',
        version: '1',
        chainId: 1,
        nonce: nonce
    });
    
    const messageToSign = siweMessage.prepareMessage();
    const signature = await wallet.signMessage(messageToSign);
    
    console.log('\nSIWE Message:', messageToSign);
    console.log('\nSignature:', signature);
    
    return { address, messageToSign, signature };
}

// If private key and nonce are provided as arguments, use them
const privateKey = process.argv[2];
const nonce = process.argv[3];
if (privateKey && nonce) {
    generateAndSignMessage(privateKey, nonce);
} 