// Client-side only wrapper for MetaMask Delegation Toolkit
// This helps avoid SSR and webpack bundling issues

let delegationToolkit: any = null;

export async function getDelegationToolkit() {
  if (typeof window === 'undefined') {
    throw new Error('MetaMask Delegation Toolkit can only be used on client side');
  }

  if (!delegationToolkit) {
    try {
      console.log('📦 Starting dynamic import of @metamask/delegation-toolkit...');
      const toolkit = await import('@metamask/delegation-toolkit');
      console.log('✅ MetaMask delegation toolkit imported successfully:', {
        exports: Object.keys(toolkit),
        hasDefault: !!toolkit.default,
        hasToMetaMaskSmartAccount: !!toolkit.toMetaMaskSmartAccount,
        hasImplementation: !!toolkit.Implementation,
      });
      
      // Handle both default and named exports
      delegationToolkit = toolkit.default || toolkit;
      
      // Verify the exports we need
      if (!delegationToolkit.toMetaMaskSmartAccount || !delegationToolkit.Implementation) {
        console.error('❌ MetaMask delegation toolkit missing required exports:', {
          hasToMetaMaskSmartAccount: !!delegationToolkit.toMetaMaskSmartAccount,
          hasImplementation: !!delegationToolkit.Implementation,
          actualExports: Object.keys(delegationToolkit),
        });
        throw new Error('MetaMask delegation toolkit missing required exports');
      }
    } catch (error) {
      console.error('Failed to load MetaMask Delegation Toolkit:', error);
      throw error;
    }
  }

  return delegationToolkit;
}

// Export a placeholder for server-side rendering
export const DelegationToolkitPlaceholder = {
  toMetaMaskSmartAccount: () => {
    throw new Error('MetaMask Delegation Toolkit not available on server side');
  },
  Implementation: {
    Hybrid: null,
  },
};