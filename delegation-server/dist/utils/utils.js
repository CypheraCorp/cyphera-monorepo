"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.customJSONStringify = exports.formatPrivateKey = void 0;
/**
 * Formats a private key to ensure it has the correct format for viem
 * @param key The private key to format
 * @returns A properly formatted private key with 0x prefix
 */
const formatPrivateKey = (key) => {
    if (key === undefined || key === null || key === "") {
        console.error("PRIVATE_KEY is required in .env file");
        process.exit(1);
    }
    // Remove any whitespace
    let formattedKey = key.trim();
    // Remove quotes if present
    if ((formattedKey.startsWith('"') && formattedKey.endsWith('"')) ||
        (formattedKey.startsWith("'") && formattedKey.endsWith("'"))) {
        formattedKey = formattedKey.slice(1, -1);
    }
    // Add 0x prefix if missing
    if (!formattedKey.startsWith('0x')) {
        formattedKey = `0x${formattedKey}`;
    }
    // Ensure it's a valid hex string
    if (!/^0x[0-9a-fA-F]+$/.test(formattedKey)) {
        throw new Error(`Invalid private key format: ${formattedKey.substring(0, 6)}...`);
    }
    // Check length - should be 66 characters (0x + 64 hex chars)
    if (formattedKey.length !== 66) {
        console.warn(`Warning: Private key has unusual length: ${formattedKey.length} (expected 66)`);
    }
    return formattedKey;
};
exports.formatPrivateKey = formatPrivateKey;
/**
 * Custom JSON serializer to handle BigInt
 */
const customJSONStringify = (obj) => {
    return JSON.stringify(obj, (_, value) => typeof value === 'bigint' ? value.toString() : value);
};
exports.customJSONStringify = customJSONStringify;
