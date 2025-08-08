'use client';

import React, { useState } from 'react';
import { ChevronDownIcon } from '@heroicons/react/24/outline';
import { logger } from '@/lib/core/logger/logger-utils';

interface NetworkConfig {
  chainId: number;
  name: string;
  displayName: string;
  color: string;
  icon?: string;
}

interface NetworkSwitcherProps {
  currentChainId: number;
  onNetworkSwitch: (chainId: number) => Promise<void>;
  isNetworkSwitching?: boolean;
  className?: string;
}

const SUPPORTED_NETWORKS: NetworkConfig[] = [
  {
    chainId: 84532,
    name: 'Base Sepolia',
    displayName: 'Base Sepolia',
    color: 'bg-blue-500',
  },
  {
    chainId: 11155111,
    name: 'Ethereum Sepolia',
    displayName: 'Ethereum Sepolia', 
    color: 'bg-gray-500',
  },
];

export const NetworkSwitcher: React.FC<NetworkSwitcherProps> = ({
  currentChainId,
  onNetworkSwitch,
  isNetworkSwitching = false,
  className = '',
}) => {
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [switchingToChain, setSwitchingToChain] = useState<number | null>(null);

  const currentNetwork = SUPPORTED_NETWORKS.find(n => n.chainId === currentChainId);
  const otherNetworks = SUPPORTED_NETWORKS.filter(n => n.chainId !== currentChainId);

  const handleNetworkSwitch = async (chainId: number) => {
    if (isNetworkSwitching || switchingToChain) return;

    try {
      logger.log('🔄 Switching to network:', { chainId });
      setSwitchingToChain(chainId);
      setIsDropdownOpen(false);
      
      await onNetworkSwitch(chainId);
      
      logger.log('✅ Network switched successfully:', { chainId });
    } catch (error) {
      logger.error('❌ Network switch failed:', error);
    } finally {
      setSwitchingToChain(null);
    }
  };

  const NetworkBadge: React.FC<{ network: NetworkConfig; isButton?: boolean; onClick?: () => void }> = ({ 
    network, 
    isButton = false, 
    onClick 
  }) => (
    <div
      className={`
        flex items-center space-x-2 px-3 py-2 rounded-lg text-sm font-medium
        ${isButton ? 'hover:bg-gray-50 cursor-pointer transition-colors' : ''}
        ${switchingToChain === network.chainId ? 'opacity-50 cursor-not-allowed' : ''}
      `}
      onClick={onClick}
    >
      <div className={`w-3 h-3 rounded-full ${network.color}`} />
      <span className="text-gray-700">{network.displayName}</span>
      {switchingToChain === network.chainId && (
        <div className="animate-spin rounded-full h-4 w-4 border-2 border-gray-300 border-t-blue-600" />
      )}
    </div>
  );

  if (!currentNetwork) {
    return (
      <div className={`bg-red-50 border border-red-200 rounded-lg p-3 ${className}`}>
        <p className="text-red-800 text-sm font-medium">⚠️ Unsupported Network</p>
        <p className="text-red-600 text-xs">Chain ID: {currentChainId}</p>
      </div>
    );
  }

  return (
    <div className={`relative ${className}`}>
      {/* Current Network Display */}
      <button
        onClick={() => setIsDropdownOpen(!isDropdownOpen)}
        disabled={isNetworkSwitching || !!switchingToChain}
        className={`
          flex items-center justify-between w-full px-4 py-3 bg-white border border-gray-200 
          rounded-lg hover:border-gray-300 transition-colors
          ${isNetworkSwitching || switchingToChain ? 'cursor-not-allowed opacity-50' : 'cursor-pointer'}
        `}
      >
        <NetworkBadge network={currentNetwork} />
        
        <div className="flex items-center space-x-2">
          {(isNetworkSwitching || !!switchingToChain) && (
            <div className="animate-spin rounded-full h-4 w-4 border-2 border-gray-300 border-t-blue-600" />
          )}
          <ChevronDownIcon 
            className={`w-4 h-4 text-gray-500 transition-transform ${isDropdownOpen ? 'rotate-180' : ''}`} 
          />
        </div>
      </button>

      {/* Network Dropdown */}
      {isDropdownOpen && (
        <>
          {/* Backdrop */}
          <div 
            className="fixed inset-0 z-10"
            onClick={() => setIsDropdownOpen(false)}
          />
          
          {/* Dropdown Menu */}
          <div className="absolute top-full left-0 right-0 mt-2 bg-white border border-gray-200 rounded-lg shadow-lg z-20">
            <div className="py-2">
              <div className="px-3 py-2 text-xs font-medium text-gray-500 uppercase tracking-wide border-b border-gray-100">
                Switch Network
              </div>
              
              {otherNetworks.map((network) => (
                <NetworkBadge
                  key={network.chainId}
                  network={network}
                  isButton={true}
                  onClick={() => handleNetworkSwitch(network.chainId)}
                />
              ))}
            </div>
          </div>
        </>
      )}

      {/* Network Switch Status */}
      {(isNetworkSwitching || !!switchingToChain) && (
        <div className="absolute top-full left-0 right-0 mt-2 bg-blue-50 border border-blue-200 rounded-lg p-3 z-20">
          <div className="flex items-center space-x-2">
            <div className="animate-spin rounded-full h-4 w-4 border-2 border-blue-300 border-t-blue-600" />
            <span className="text-blue-800 text-sm font-medium">
              {switchingToChain 
                ? `Switching to ${SUPPORTED_NETWORKS.find(n => n.chainId === switchingToChain)?.displayName}...`
                : 'Switching network...'
              }
            </span>
          </div>
        </div>
      )}
    </div>
  );
};