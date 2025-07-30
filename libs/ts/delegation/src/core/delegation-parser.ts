import { Delegation } from '@metamask/delegation-toolkit';

/**
 * Parse a delegation from either bytes or JSON format
 * PRESERVED from apps/delegation-server/src/utils/delegation-helpers.ts
 * @param delegationData The delegation data as either Uint8Array or Buffer
 * @returns The parsed delegation structure
 */
export function parseDelegation(delegationData: Uint8Array | Buffer): Delegation {
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
      return parsedObject as Delegation;
    } catch (jsonError) {
      console.debug('Failed to parse delegation as JSON, trying alternative format', jsonError);
      
      // If JSON parsing fails, the delegation might be in a different format
      // In a real implementation, we would attempt other parsing methods here
      // However, since we don't have direct access to DelegationFramework.decode,
      // we'll throw a more descriptive error
      
      throw new Error('Failed to parse delegation: Binary format not supported in this version');
    }
  } catch (error) {
    console.error('Failed to parse delegation data', error);
    throw new Error(`Failed to parse delegation: ${error instanceof Error ? error.message : String(error)}`);
  }
}