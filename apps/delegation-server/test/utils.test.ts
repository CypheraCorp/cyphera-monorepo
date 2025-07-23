/**
 * Unit tests for utility functions
 * 
 * To run:
 *   npx jest test/utils.test.ts
 */

import { 
  formatPrivateKey, 
  customJSONStringify, 
  safeJsonParse, 
  bytesToHex, 
  hexToBytes 
} from '../src/utils/utils'

// Import Jest functions
import { describe, expect, it } from '@jest/globals';

describe('Utility Functions', () => {
  describe('formatPrivateKey', () => {
    it('should add 0x prefix if missing', () => {
      const privateKey = '1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef'
      expect(formatPrivateKey(privateKey)).toBe(`0x${privateKey}`)
    })

    it('should keep 0x prefix if present', () => {
      const privateKey = '0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef'
      expect(formatPrivateKey(privateKey)).toBe(privateKey)
    })

    it('should throw error if key is wrong length', () => {
      const privateKey = '1234567890abcdef'
      expect(() => formatPrivateKey(privateKey)).toThrow('Invalid private key length')
    })

    it('should throw error if key contains non-hex characters', () => {
      const privateKey = '1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdez'
      expect(() => formatPrivateKey(privateKey)).toThrow('Invalid private key format')
    })

    it('should throw error if key is empty', () => {
      expect(() => formatPrivateKey('')).toThrow('Private key is required')
    })
  })

  describe('customJSONStringify', () => {
    it('should convert BigInt to string', () => {
      const obj = { value: BigInt(123) }
      expect(customJSONStringify(obj)).toBe('{"value":"123"}')
    })

    it('should handle regular JSON', () => {
      const obj = { a: 1, b: 'string', c: true }
      expect(customJSONStringify(obj)).toBe('{"a":1,"b":"string","c":true}')
    })

    it('should handle nested objects with BigInt', () => {
      const obj = { 
        a: 1, 
        b: { 
          c: BigInt(123),
          d: 'test'
        } 
      }
      expect(customJSONStringify(obj)).toBe('{"a":1,"b":{"c":"123","d":"test"}}')
    })
  })

  describe('safeJsonParse', () => {
    it('should parse valid JSON', () => {
      const json = '{"a":1,"b":"string"}'
      expect(safeJsonParse(json)).toEqual({ a: 1, b: 'string' })
    })

    it('should return fallback for invalid JSON', () => {
      const invalidJson = '{a:1'
      expect(safeJsonParse(invalidJson, null)).toBeNull()
    })

    it('should return fallback for empty string', () => {
      expect(safeJsonParse('', { empty: true })).toEqual({ empty: true })
    })
  })

  describe('bytesToHex and hexToBytes', () => {
    it('should convert bytes to hex string', () => {
      const bytes = new Uint8Array([0x12, 0x34, 0xab, 0xcd])
      expect(bytesToHex(bytes)).toBe('0x1234abcd')
    })

    it('should convert hex string to bytes', () => {
      const hex = '0x1234abcd'
      const bytes = hexToBytes(hex)
      expect(bytes).toEqual(new Uint8Array([0x12, 0x34, 0xab, 0xcd]))
    })

    it('should handle hex string without 0x prefix', () => {
      const hex = '1234abcd'
      const bytes = hexToBytes(hex)
      expect(bytes).toEqual(new Uint8Array([0x12, 0x34, 0xab, 0xcd]))
    })

    it('should convert bytes to hex and back', () => {
      const original = new Uint8Array([0x12, 0x34, 0xab, 0xcd])
      const hex = bytesToHex(original)
      const result = hexToBytes(hex)
      expect(result).toEqual(original)
    })
  })
}) 