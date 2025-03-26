"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.isValidEthereumAddress = isValidEthereumAddress;
exports.parseDelegation = parseDelegation;
exports.validateDelegation = validateDelegation;
const utils_1 = require("./utils");
/**
 * Validates that the input is a valid Ethereum address
 * @param address The address to validate
 * @returns true if valid, throws error if invalid
 */
function isValidEthereumAddress(address) {
    // Check if address is defined
    if (!address) {
        return false;
    }
    // Check if address starts with 0x
    if (!address.startsWith('0x')) {
        return false;
    }
    // Check if address is 42 characters (0x + 40 hex digits)
    if (address.length !== 42) {
        return false;
    }
    // Check if address contains only hexadecimal characters after 0x
    const hexPart = address.slice(2);
    if (!/^[0-9a-fA-F]{40}$/.test(hexPart)) {
        return false;
    }
    return true;
}
/**
 * Parse a delegation from either bytes or JSON format
 * @param delegationData The delegation data as either Uint8Array or Buffer
 * @returns The parsed delegation structure
 */
function parseDelegation(delegationData) {
    try {
        try {
            // Try to parse as JSON first (most common case in our system)
            const jsonString = Buffer.from(delegationData).toString('utf-8');
            const parsedObject = JSON.parse(jsonString);
            // Validate the parsed object has the required fields
            if (!parsedObject.delegator || !parsedObject.delegate) {
                throw new Error('Invalid delegation format: missing required fields');
            }
            // If parsed successfully, return the object
            return parsedObject;
        }
        catch (jsonError) {
            utils_1.logger.debug('Failed to parse delegation as JSON, trying alternative format', jsonError);
            // If JSON parsing fails, the delegation might be in a different format
            // In a real implementation, we would attempt other parsing methods here
            // However, since we don't have direct access to DelegationFramework.decode,
            // we'll throw a more descriptive error
            throw new Error('Failed to parse delegation: Binary format not supported in this version');
        }
    }
    catch (error) {
        utils_1.logger.error('Failed to parse delegation data', error);
        throw new Error(`Failed to parse delegation: ${error instanceof Error ? error.message : String(error)}`);
    }
}
/**
 * Validates a delegation structure
 * @param delegation The delegation to validate
 * @returns true if valid, throws error if invalid
 */
function validateDelegation(delegation) {
    // Check required fields
    if (!delegation.delegator) {
        throw new Error('Invalid delegation: missing delegator');
    }
    if (!delegation.delegate) {
        throw new Error('Invalid delegation: missing delegate');
    }
    if (!delegation.signature) {
        throw new Error('Invalid delegation: missing signature');
    }
    // Validate delegator address format
    if (!isValidEthereumAddress(delegation.delegator)) {
        throw new Error('Invalid delegator address format: must be a valid Ethereum address (0x + 40 hex chars)');
    }
    // Validate delegate address format
    if (!isValidEthereumAddress(delegation.delegate)) {
        throw new Error('Invalid delegate address format: must be a valid Ethereum address (0x + 40 hex chars)');
    }
    // Check if delegation is expired (convert various types to bigint for consistent comparison)
    let expiryBigInt = null;
    if (delegation.expiry) {
        if (typeof delegation.expiry === 'bigint') {
            expiryBigInt = delegation.expiry;
        }
        else if (typeof delegation.expiry === 'number') {
            expiryBigInt = BigInt(delegation.expiry);
        }
        else if (typeof delegation.expiry === 'string') {
            try {
                expiryBigInt = BigInt(delegation.expiry);
            }
            catch (error) {
                utils_1.logger.error('Invalid expiry format', error);
                throw new Error('Invalid expiry format: must be a valid number or bigint');
            }
        }
        // Check if the delegation is expired
        if (expiryBigInt !== null && expiryBigInt > 0n) {
            const nowBigInt = BigInt(Math.floor(Date.now() / 1000));
            if (expiryBigInt < nowBigInt) {
                throw new Error(`Delegation is expired (expiry: ${expiryBigInt}, now: ${nowBigInt})`);
            }
        }
    }
    return true;
}
