"use strict";
/**
 * Unit tests for utility functions
 *
 * To run:
 *   npx jest test/utils.test.ts
 */
Object.defineProperty(exports, "__esModule", { value: true });
const utils_1 = require("../src/utils/utils");
// Import Jest functions
const globals_1 = require("@jest/globals");
(0, globals_1.describe)('Utility Functions', () => {
    (0, globals_1.describe)('formatPrivateKey', () => {
        (0, globals_1.it)('should add 0x prefix if missing', () => {
            const privateKey = '1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef';
            (0, globals_1.expect)((0, utils_1.formatPrivateKey)(privateKey)).toBe(`0x${privateKey}`);
        });
        (0, globals_1.it)('should keep 0x prefix if present', () => {
            const privateKey = '0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef';
            (0, globals_1.expect)((0, utils_1.formatPrivateKey)(privateKey)).toBe(privateKey);
        });
        (0, globals_1.it)('should throw error if key is wrong length', () => {
            const privateKey = '1234567890abcdef';
            (0, globals_1.expect)(() => (0, utils_1.formatPrivateKey)(privateKey)).toThrow('Invalid private key length');
        });
        (0, globals_1.it)('should throw error if key contains non-hex characters', () => {
            const privateKey = '1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdez';
            (0, globals_1.expect)(() => (0, utils_1.formatPrivateKey)(privateKey)).toThrow('Invalid private key format');
        });
        (0, globals_1.it)('should throw error if key is empty', () => {
            (0, globals_1.expect)(() => (0, utils_1.formatPrivateKey)('')).toThrow('Private key is required');
        });
    });
    (0, globals_1.describe)('customJSONStringify', () => {
        (0, globals_1.it)('should convert BigInt to string', () => {
            const obj = { value: BigInt(123) };
            (0, globals_1.expect)((0, utils_1.customJSONStringify)(obj)).toBe('{"value":"123"}');
        });
        (0, globals_1.it)('should handle regular JSON', () => {
            const obj = { a: 1, b: 'string', c: true };
            (0, globals_1.expect)((0, utils_1.customJSONStringify)(obj)).toBe('{"a":1,"b":"string","c":true}');
        });
        (0, globals_1.it)('should handle nested objects with BigInt', () => {
            const obj = {
                a: 1,
                b: {
                    c: BigInt(123),
                    d: 'test'
                }
            };
            (0, globals_1.expect)((0, utils_1.customJSONStringify)(obj)).toBe('{"a":1,"b":{"c":"123","d":"test"}}');
        });
    });
    (0, globals_1.describe)('safeJsonParse', () => {
        (0, globals_1.it)('should parse valid JSON', () => {
            const json = '{"a":1,"b":"string"}';
            (0, globals_1.expect)((0, utils_1.safeJsonParse)(json)).toEqual({ a: 1, b: 'string' });
        });
        (0, globals_1.it)('should return fallback for invalid JSON', () => {
            const invalidJson = '{a:1';
            (0, globals_1.expect)((0, utils_1.safeJsonParse)(invalidJson, null)).toBeNull();
        });
        (0, globals_1.it)('should return fallback for empty string', () => {
            (0, globals_1.expect)((0, utils_1.safeJsonParse)('', { empty: true })).toEqual({ empty: true });
        });
    });
    (0, globals_1.describe)('bytesToHex and hexToBytes', () => {
        (0, globals_1.it)('should convert bytes to hex string', () => {
            const bytes = new Uint8Array([0x12, 0x34, 0xab, 0xcd]);
            (0, globals_1.expect)((0, utils_1.bytesToHex)(bytes)).toBe('0x1234abcd');
        });
        (0, globals_1.it)('should convert hex string to bytes', () => {
            const hex = '0x1234abcd';
            const bytes = (0, utils_1.hexToBytes)(hex);
            (0, globals_1.expect)(bytes).toEqual(new Uint8Array([0x12, 0x34, 0xab, 0xcd]));
        });
        (0, globals_1.it)('should handle hex string without 0x prefix', () => {
            const hex = '1234abcd';
            const bytes = (0, utils_1.hexToBytes)(hex);
            (0, globals_1.expect)(bytes).toEqual(new Uint8Array([0x12, 0x34, 0xab, 0xcd]));
        });
        (0, globals_1.it)('should convert bytes to hex and back', () => {
            const original = new Uint8Array([0x12, 0x34, 0xab, 0xcd]);
            const hex = (0, utils_1.bytesToHex)(original);
            const result = (0, utils_1.hexToBytes)(hex);
            (0, globals_1.expect)(result).toEqual(original);
        });
    });
});
