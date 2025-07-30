import { type Hex, toHex } from 'viem';

/**
 * Crypto utilities for delegation operations
 */

/**
 * Generates a cryptographically secure random salt
 * @param length The length of the salt in bytes (default: 8)
 * @returns A hex string representing the salt
 */
export function generateSecureSalt(length: number = 8): Hex {
  // Check if we're in a browser environment with crypto API
  if (typeof window !== 'undefined' && window.crypto && window.crypto.getRandomValues) {
    const array = new Uint8Array(length);
    window.crypto.getRandomValues(array);
    return toHex(array);
  }
  
  // Check if we're in Node.js environment
  if (typeof require !== 'undefined') {
    try {
      const crypto = require('crypto');
      const buffer = crypto.randomBytes(length);
      return toHex(buffer);
    } catch (error) {
      // Fall back to timestamp-based generation if crypto module is not available
    }
  }
  
  // Fallback for environments without secure random generation
  console.warn('Secure random generation not available, falling back to timestamp-based salt');
  return generateTimestampBasedSalt();
}

/**
 * Generates a timestamp-based salt (less secure, for fallback only)
 * @returns A hex string representing the salt
 */
function generateTimestampBasedSalt(): Hex {
  const timestamp = Date.now().toString();
  const random = Math.random().toString();
  const combined = timestamp + random;
  
  // Create a simple hash of the combined string
  let hash = 0;
  for (let i = 0; i < combined.length; i++) {
    hash = (hash << 5) - hash + combined.charCodeAt(i);
    hash = hash & hash; // Convert to 32bit integer
  }
  
  // Convert to hex string of appropriate length
  const hexHash = Math.abs(hash).toString(16).padStart(16, '0');
  return `0x${hexHash}` as Hex;
}

/**
 * Validates if a string is a valid hex string
 * @param value The string to validate
 * @returns True if valid hex string, false otherwise
 */
export function isValidHex(value: string): boolean {
  if (!value.startsWith('0x')) {
    return false;
  }
  
  const hexPart = value.slice(2);
  return /^[0-9a-fA-F]*$/.test(hexPart);
}

/**
 * Validates if a string is a valid Ethereum address
 * @param address The address to validate
 * @returns True if valid address, false otherwise
 */
export function isValidAddress(address: string): boolean {
  if (!address) {
    return false;
  }
  
  if (!address.startsWith('0x')) {
    return false;
  }
  
  if (address.length !== 42) {
    return false;
  }
  
  const hexPart = address.slice(2);
  return /^[0-9a-fA-F]{40}$/.test(hexPart);
}