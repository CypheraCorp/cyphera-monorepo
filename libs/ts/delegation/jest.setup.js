// Jest setup file for delegation library
// This file runs before each test file

// Mock window.crypto for tests that don't run in browser environment
global.crypto = {
  getRandomValues: (arr) => {
    for (let i = 0; i < arr.length; i++) {
      arr[i] = Math.floor(Math.random() * 256);
    }
    return arr;
  },
};

// Mock MetaMask delegation toolkit if needed
jest.mock('@metamask/delegation-toolkit', () => ({
  createDelegation: jest.fn(),
  Delegation: {},
  Caveat: {},
}));