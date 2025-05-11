import { toMetaMaskSmartAccount, Implementation, type MetaMaskSmartAccount } from "@metamask/delegation-toolkit";
import { createPublicClient, http, formatEther, isAddress, type Address, parseEther } from "viem";
import { privateKeyToAccount, type LocalAccount } from "viem/accounts";
import { baseSepolia, sepolia, type Chain } from "viem/chains";
import dotenv from "dotenv";
import { createBundlerClient } from "viem/account-abstraction";
import { createPimlicoClient } from "permissionless/clients/pimlico";

// Note: The ABI is in a .ts file. If your environment doesn't handle this directly when using require,
// you might need to compile it to .js first or use a setup that supports direct .ts import in .js.
// For simplicity, this script assumes it can be required. Alternatively, you can paste the ABI array directly.
// const { simpleFactoryAbi } = require("../delegation-server/src/abis/simpleFactory.ts");

dotenv.config();

// --- Configuration ---
const DEPLOYER_PRIVATE_KEY_ENV: string | undefined = process.env.PRIVATE_KEY;
const INFURA_API_KEY: string | undefined = process.env.INFURA_API_KEY;
const PIMLICO_API_KEY: string | undefined = process.env.PIMLICO_API_KEY;
const SALT: `0x${string}` = "0x";

// --- Target Network ---
const TARGET_NETWORK_NAME: string = "BASE_SEPOLIA";

if (!DEPLOYER_PRIVATE_KEY_ENV) {
  console.error("Error: PRIVATE_KEY environment variable is not set.");
  process.exit(1);
}
// Cast to string after check, as validatePrivateKey expects string
const DEPLOYER_PRIVATE_KEY: string = DEPLOYER_PRIVATE_KEY_ENV;

if (!INFURA_API_KEY) {
  console.warn("Warning: INFURA_API_KEY environment variable is not set. Using default public RPCs.");
}
if (!PIMLICO_API_KEY) {
  console.error("Error: PIMLICO_API_KEY environment variable is not set. Bundler operations will fail.");
  process.exit(1);
}

const baseSepoliaRpc = INFURA_API_KEY ? `https://base-sepolia.infura.io/v3/${INFURA_API_KEY}` : "https://sepolia.base.org";
const ethSepoliaRpc = INFURA_API_KEY ? `https://sepolia.infura.io/v3/${INFURA_API_KEY}` : "https://rpc.sepolia.org";

// Construct Bundler URLs with Pimlico API Key
const baseSepoliaBundlerUrl = `https://api.pimlico.io/v1/base-sepolia/rpc?apikey=${PIMLICO_API_KEY}`;
const ethSepoliaBundlerUrl = `https://api.pimlico.io/v1/sepolia/rpc?apikey=${PIMLICO_API_KEY}`;

interface NetworkConfig {
  chainId: number;
  rpcUrl: string;
  bundlerUrl: string;
  simpleFactoryAddress: Address;
  chain: Chain;
  name: string;
  explorerUrl: string;
}

const NETWORKS: Record<string, NetworkConfig> = {
  BASE_SEPOLIA: {
    chainId: 84532,
    rpcUrl: baseSepoliaRpc,
    bundlerUrl: baseSepoliaBundlerUrl,
    simpleFactoryAddress: "0x69Aa2f9fe1572F1B640E1bbc512f5c3a734fc77c" as Address,
    chain: baseSepolia,
    name: "Base Sepolia",
    explorerUrl: "https://sepolia.basescan.org/address/",
  },
  ETH_SEPOLIA: {
    chainId: 11155111,
    rpcUrl: ethSepoliaRpc,
    bundlerUrl: ethSepoliaBundlerUrl,
    simpleFactoryAddress: "0x69Aa2f9fe1572F1B640E1bbc512f5c3a734fc77c" as Address,
    chain: sepolia,
    name: "Ethereum Sepolia",
    explorerUrl: "https://sepolia.etherscan.io/address/",
  },
  // Add more network configurations here
  // EXAMPLE_NETWORK: {
  //   chainId: 12345,
  //   rpcUrl: "YOUR_RPC_URL",
  //   simpleFactoryAddress: "YOUR_FACTORY_ADDRESS_ON_THIS_NETWORK",
  //   chain: { /* viem chain object */ id: 12345, name: 'Example', nativeCurrency: { name: 'Example Ether', symbol: 'ETH', decimals: 18 } },
  //   name: "Example Network",
  //   explorerUrl: "YOUR_EXPLORER_URL_PREFIX/",
  // }
};

// --- Helper Function ---
function validatePrivateKey(privateKey: string): `0x${string}` | false {
  // privateKey is guaranteed to be a string here due to checks above
  let keyToValidate = privateKey;
  if (!keyToValidate.startsWith("0x")) {
    keyToValidate = `0x${keyToValidate}`;
  }
  if (keyToValidate.length !== 66) {
    console.error(`Error: Invalid private key length. Expected 66 hex chars, got ${keyToValidate.length}.`);
    return false;
  }
  return keyToValidate as `0x${string}`;
}

// --- Main Deployment Logic ---
async function deploySmartAccount() {
  console.log(`Attempting to deploy Smart Account on ${TARGET_NETWORK_NAME}...`);

  const validatedPrivateKey = validatePrivateKey(DEPLOYER_PRIVATE_KEY);
  if (!validatedPrivateKey) process.exit(1);

  const networkConfig = NETWORKS[TARGET_NETWORK_NAME];
  if (!networkConfig) {
    console.error(`Error: Network configuration for "${TARGET_NETWORK_NAME}" not found.`);
    process.exit(1);
  }
  
  if (!isAddress(networkConfig.simpleFactoryAddress)) {
    console.error(`Error: simpleFactoryAddress is not a valid address for ${TARGET_NETWORK_NAME}.`);
    process.exit(1);
  }

  try {
    const deployerAccount: LocalAccount<string, `0x${string}`> = privateKeyToAccount(validatedPrivateKey);
    console.log(`Using deployer account: ${deployerAccount.address}`);

    const publicClient = createPublicClient({
      chain: networkConfig.chain,
      transport: http(networkConfig.rpcUrl),
    });

    const pimlicoClient = createPimlicoClient({
        chain: networkConfig.chain,
        transport: http(networkConfig.bundlerUrl),
    });

    const bundlerClient = createBundlerClient({
        chain: networkConfig.chain,
        transport: http(networkConfig.bundlerUrl),
    });

    console.log(`Checking balance for ${deployerAccount.address} on ${networkConfig.name}...`);
    const balance = await publicClient.getBalance({ address: deployerAccount.address });
    console.log("deployer raw balance", balance);
    console.log(`Deployer balance: ${formatEther(balance)} ${networkConfig.chain.nativeCurrency.symbol}`);

    if (balance === 0n) {
      console.warn(`Warning: Deployer account ${deployerAccount.address} has zero balance. Deployment will likely fail.`);
    }

    console.log(`Initializing MetaMask Hybrid Smart Account...`);

    const smartAccount = await toMetaMaskSmartAccount({
      client: publicClient,
      implementation: Implementation.Hybrid,
      signatory: { account: deployerAccount },
      deploySalt: SALT,
      deployParams: [
        deployerAccount.address,
        [],
        [],
        []
      ],
    }) as MetaMaskSmartAccount<Implementation.Hybrid>;

    const isDeployed = await smartAccount.isDeployed();
    console.log(isDeployed ? "Smart Account is already deployed" : "Smart Account is not deployed");

    if (!isDeployed) {
      console.log("Smart account not deployed. Sending dummy transaction to trigger deployment...");
      
      const { fast: gasPrices } = await pimlicoClient.getUserOperationGasPrice();
      if (!gasPrices || !gasPrices.maxFeePerGas || !gasPrices.maxPriorityFeePerGas) {
        console.error("Error: Could not fetch gas prices from Pimlico.");
        process.exit(1);
      }
      console.log(`Using gas prices: maxFeePerGas: ${gasPrices.maxFeePerGas}, maxPriorityFeePerGas: ${gasPrices.maxPriorityFeePerGas}`);

      const DUMMY_RECEIVER_ADDRESS = "0x819f58bEf39B809B34e9Cce706CEf95476035136";
      const DUMMY_VALUE = parseEther("0.000001");

      console.log(`Sending ${formatEther(DUMMY_VALUE)} ETH to ${DUMMY_RECEIVER_ADDRESS} to deploy SA ${smartAccount.address}`);

      const userOperationHash = await bundlerClient.sendUserOperation({
        account: smartAccount,
        calls: [
          {
            to: DUMMY_RECEIVER_ADDRESS as Address,
            value: DUMMY_VALUE,
          }
        ],
        maxFeePerGas: gasPrices.maxFeePerGas,
        maxPriorityFeePerGas: gasPrices.maxPriorityFeePerGas,
        verificationGasLimit: 150000n,
      });

      console.log("UserOperation sent. Hash:", userOperationHash);
      console.log("Waiting for UserOperation receipt...");

      const receipt = await bundlerClient.waitForUserOperationReceipt({
        hash: userOperationHash,
        timeout: 60000,
      });

      console.log("UserOperation successful! Transaction Hash:", receipt.receipt.transactionHash);
      console.log(`Smart Account ${smartAccount.address} should now be deployed.`);
    }

    console.log(`Smart Account Address: ${smartAccount.address}`);
    if (networkConfig.explorerUrl) {
      console.log(`View on explorer: ${networkConfig.explorerUrl}${smartAccount.address}`);
    }

    const code = await publicClient.getBytecode({ address: smartAccount.address });
    if (code && code !== "0x") {
      console.log(`Smart Account bytecode successfully found on-chain at ${smartAccount.address}. Deployment confirmed!`);
    } else {
      console.warn(`Warning: Smart Account bytecode not found at ${smartAccount.address}. This is unexpected if a deployment transaction was sent.`);
    }

  } catch (error: unknown) {
    console.error("Error in deploySmartAccount script:", error);
    if (error instanceof Error) {
      if (error.message?.includes("insufficient funds")) {
        console.error("This error often indicates the deployer account does not have enough native currency for gas.");
      }
    } else {
      console.error("An unexpected error type occurred:", error);
    }
    process.exit(1);
  }

  // Exit with success code
  process.exit(0);
}

// --- Run the script ---
deploySmartAccount(); 