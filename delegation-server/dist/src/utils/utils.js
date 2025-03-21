"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.logger = exports.customJSONStringify = void 0;
exports.formatPrivateKey = formatPrivateKey;
exports.safeJsonParse = safeJsonParse;
exports.bytesToHex = bytesToHex;
exports.hexToBytes = hexToBytes;
/**
 * Formats a private key to ensure it has the 0x prefix
 */
function formatPrivateKey(privateKey) {
    if (!privateKey) {
        throw new Error('Private key is required');
    }
    // Remove 0x prefix if it exists
    const cleanKey = privateKey.startsWith('0x') ? privateKey.slice(2) : privateKey;
    // Validate key length (32 bytes = 64 hex characters)
    if (cleanKey.length !== 64) {
        throw new Error(`Invalid private key length: expected 64 hex characters, got ${cleanKey.length}`);
    }
    // Check if key contains only valid hex characters
    if (!/^[0-9a-fA-F]{64}$/.test(cleanKey)) {
        throw new Error('Invalid private key format: must contain only hex characters');
    }
    // Return with 0x prefix
    return `0x${cleanKey}`;
}
/**
 * Custom JSON serializer to handle BigInt
 */
const customJSONStringify = (obj) => {
    return JSON.stringify(obj, (_, value) => typeof value === 'bigint' ? value.toString() : value);
};
exports.customJSONStringify = customJSONStringify;
/**
 * Simple logging utility with timestamp
 */
exports.logger = {
    debug: (message, ...args) => {
        if (process.env.LOG_LEVEL === 'debug') {
            console.debug(`[${new Date().toISOString()}] DEBUG: ${message}`, ...args);
        }
    },
    info: (message, ...args) => {
        console.info(`[${new Date().toISOString()}] INFO: ${message}`, ...args);
    },
    warn: (message, ...args) => {
        console.warn(`[${new Date().toISOString()}] WARN: ${message}`, ...args);
    },
    error: (message, ...args) => {
        console.error(`[${new Date().toISOString()}] ERROR: ${message}`, ...args);
    }
};
/**
 * Safely parse JSON without throwing
 */
function safeJsonParse(str, fallback = null) {
    try {
        return JSON.parse(str);
    }
    catch (error) {
        return fallback;
    }
}
/**
 * Convert bytes to hex string
 */
function bytesToHex(bytes) {
    return '0x' + Array.from(bytes)
        .map(b => b.toString(16).padStart(2, '0'))
        .join('');
}
/**
 * Convert hex string to bytes
 */
function hexToBytes(hex) {
    const cleanHex = hex.startsWith('0x') ? hex.slice(2) : hex;
    const bytes = new Uint8Array(cleanHex.length / 2);
    for (let i = 0; i < cleanHex.length; i += 2) {
        bytes[i / 2] = parseInt(cleanHex.slice(i, i + 2), 16);
    }
    return bytes;
}
