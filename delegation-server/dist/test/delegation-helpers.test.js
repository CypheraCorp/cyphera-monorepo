"use strict";
/**
 * Unit tests for delegation helper functions
 *
 * To run:
 *   npx jest test/delegation-helpers.test.ts
 */
Object.defineProperty(exports, "__esModule", { value: true });
const delegation_helpers_1 = require("../src/utils/delegation-helpers");
const globals_1 = require("@jest/globals");
(0, globals_1.describe)('Delegation Helper Functions', () => {
    (0, globals_1.describe)('parseDelegation', () => {
        (0, globals_1.it)('should parse valid JSON delegation data', () => {
            const mockDelegation = {
                delegator: '0x1234567890123456789012345678901234567890',
                delegate: '0x0987654321098765432109876543210987654321',
                authority: {
                    scheme: '0x00',
                    signature: '0xsig',
                    signer: '0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789'
                },
                caveats: [],
                salt: '0x123456789',
                signature: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890'
            };
            const jsonData = JSON.stringify(mockDelegation);
            const buffer = Buffer.from(jsonData, 'utf-8');
            const result = (0, delegation_helpers_1.parseDelegation)(buffer);
            (0, globals_1.expect)(result).toEqual(mockDelegation);
        });
        (0, globals_1.it)('should throw error for invalid JSON', () => {
            const buffer = Buffer.from('{invalid:json}', 'utf-8');
            (0, globals_1.expect)(() => (0, delegation_helpers_1.parseDelegation)(buffer)).toThrow('Failed to parse delegation');
        });
        (0, globals_1.it)('should throw error for missing required fields', () => {
            const jsonData = JSON.stringify({});
            const buffer = Buffer.from(jsonData, 'utf-8');
            (0, globals_1.expect)(() => (0, delegation_helpers_1.parseDelegation)(buffer)).toThrow(/Failed to parse delegation.*Binary format not supported/);
        });
    });
    (0, globals_1.describe)('validateDelegation', () => {
        (0, globals_1.it)('should validate a complete delegation object', () => {
            const mockDelegation = {
                delegator: '0x1234567890123456789012345678901234567890',
                delegate: '0x0987654321098765432109876543210987654321',
                authority: {
                    scheme: '0x00',
                    signature: '0xsig',
                    signer: '0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789'
                },
                caveats: [],
                salt: '0x123456789',
                signature: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
                scheme: '0x00',
                invocations: [],
            };
            (0, globals_1.expect)((0, delegation_helpers_1.validateDelegation)(mockDelegation)).toBe(true);
        });
        (0, globals_1.it)('should throw for missing delegator', () => {
            const mockDelegation = {
                delegate: '0x0987654321098765432109876543210987654321',
                signature: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
            };
            (0, globals_1.expect)(() => (0, delegation_helpers_1.validateDelegation)(mockDelegation)).toThrow('missing delegator');
        });
        (0, globals_1.it)('should throw for missing delegate', () => {
            const mockDelegation = {
                delegator: '0x1234567890123456789012345678901234567890',
                signature: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
            };
            (0, globals_1.expect)(() => (0, delegation_helpers_1.validateDelegation)(mockDelegation)).toThrow('missing delegate');
        });
        (0, globals_1.it)('should throw for missing signature', () => {
            const mockDelegation = {
                delegator: '0x1234567890123456789012345678901234567890',
                delegate: '0x0987654321098765432109876543210987654321',
            };
            (0, globals_1.expect)(() => (0, delegation_helpers_1.validateDelegation)(mockDelegation)).toThrow('missing signature');
        });
        (0, globals_1.it)('should throw for expired delegation', () => {
            const mockDelegation = {
                delegator: '0x1234567890123456789012345678901234567890',
                delegate: '0x0987654321098765432109876543210987654321',
                authority: {
                    scheme: '0x00',
                    signature: '0xsig',
                    signer: '0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789'
                },
                caveats: [],
                salt: '0x123456789',
                signature: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
                scheme: '0x00',
                invocations: [],
            };
            (0, globals_1.expect)(() => (0, delegation_helpers_1.validateDelegation)(mockDelegation)).toThrow('Delegation is expired');
        });
    });
});
