/**
 * Formats a private key to ensure it has the 0x prefix
 */
export declare function formatPrivateKey(privateKey: string): string;
/**
 * Custom JSON serializer to handle BigInt
 */
export declare const customJSONStringify: (obj: any) => string;
/**
 * Simple logging utility with timestamp
 */
export declare const logger: {
    debug: (message: string, ...args: any[]) => void;
    info: (message: string, ...args: any[]) => void;
    warn: (message: string, ...args: any[]) => void;
    error: (message: string, ...args: any[]) => void;
};
/**
 * Safely parse JSON without throwing
 */
export declare function safeJsonParse(str: string, fallback?: any): any;
/**
 * Convert bytes to hex string
 */
export declare function bytesToHex(bytes: Uint8Array): string;
/**
 * Convert hex string to bytes
 */
export declare function hexToBytes(hex: string): Uint8Array;
