'use client';

import React, { useEffect, useState } from 'react';
import { getDelegationToolkit } from '@/lib/web3/delegation-toolkit-wrapper';

export default function TestDelegationToolkit() {
  const [status, setStatus] = useState<string>('Loading...');
  const [error, setError] = useState<string | null>(null);
  const [exports, setExports] = useState<any>(null);

  useEffect(() => {
    const testToolkit = async () => {
      try {
        console.log('Starting delegation toolkit test...');
        const toolkit = await getDelegationToolkit();
        
        console.log('Toolkit loaded:', toolkit);
        
        const exportInfo = {
          hasToMetaMaskSmartAccount: !!toolkit.toMetaMaskSmartAccount,
          hasImplementation: !!toolkit.Implementation,
          implementationKeys: toolkit.Implementation ? Object.keys(toolkit.Implementation) : [],
          allExports: Object.keys(toolkit),
        };
        
        setExports(exportInfo);
        setStatus('Delegation toolkit loaded successfully!');
      } catch (err) {
        console.error('Failed to load toolkit:', err);
        setError(err instanceof Error ? err.message : 'Unknown error');
        setStatus('Failed to load delegation toolkit');
      }
    };

    testToolkit();
  }, []);

  return (
    <div className="p-8">
      <h1 className="text-2xl font-bold mb-4">MetaMask Delegation Toolkit Test</h1>
      
      <div className="mb-4">
        <p className="font-semibold">Status: {status}</p>
        {error && <p className="text-red-600">Error: {error}</p>}
      </div>

      {exports && (
        <div className="bg-gray-100 p-4 rounded">
          <h2 className="font-semibold mb-2">Toolkit Exports:</h2>
          <pre className="text-sm">{JSON.stringify(exports, null, 2)}</pre>
        </div>
      )}
    </div>
  );
}