import { delegationService } from '../src/services/service'
import { ServerUnaryCall, sendUnaryData } from '@grpc/grpc-js'
import config from '../src/config' // Import config to potentially mock it

// Mock the logger
jest.mock('../src/utils/utils', () => ({
  logger: {
    info: jest.fn(),
    error: jest.fn(),
    warn: jest.fn(),
  },
}))

// --- Mocking the dynamically imported functions ---
// We need to mock the *modules* that service.ts tries to import.
// We'll store mock functions here to control their behavior in tests.
let mockRedeemImplementation: jest.Mock = jest.fn()
let mockModeRedeemImplementation: jest.Mock = jest.fn(); // Separate mock for explicit mock mode test

jest.mock('../src/services/redeem-delegation', () => ({
  // This function will be returned when service.ts dynamically imports the real module
  redeemDelegation: (...args: any[]) => mockRedeemImplementation(...args),
}), { virtual: true }) // virtual: true helps if the path resolution is tricky or file doesn't exist in test env

jest.mock('../src/services/mock-redeem-delegation', () => ({
  // This function will be returned when service.ts dynamically imports the mock module
  redeemDelegation: (...args: any[]) => mockModeRedeemImplementation(...args), // Use the mock mode specific mock
}), { virtual: true })
// --- End Mocking ---


// Helper to create a mock gRPC call object
const createMockCall = (requestData: any): ServerUnaryCall<any, any> => {
  return {
    request: requestData,
    // Add other ServerUnaryCall properties/methods if needed by the service logic
  } as unknown as ServerUnaryCall<any, any>
}

// Helper to create a mock gRPC callback function
const createMockCallback = (): jest.MockedFunction<sendUnaryData<any>> => {
  return jest.fn()
}

// Define mock constants for new parameters
const mockTokenDecimals = 6;
const mockChainId = 11155111; // Sepolia
const mockNetworkName = 'ethereum sepolia';

describe('delegationService', () => {
  let originalMockMode: boolean | undefined
  let delegationService: any;
  let currentMockImpl: jest.Mock;

  // Back up original config state
  beforeAll(() => {
    originalMockMode = config.mockMode
  })

  // Reset mocks and config before each test
  beforeEach(() => {
    jest.clearAllMocks()
    // Reset the implementation mocks for each test
    mockRedeemImplementation = jest.fn()
    mockModeRedeemImplementation = jest.fn(); // Reset mock mode mock as well
     // Reset config.mockMode to its original state before each test
     // Note: Modifying config directly might not work as expected if it was already read by service.ts
     // A cleaner approach might involve jest.resetModules() and re-mocking config,
     // but let's try direct modification first.
    config.mockMode = originalMockMode! // Restore original mode

    // IMPORTANT: Due to the async dynamic import in service.ts, changing config.mockMode
    // *after* the service module has initially loaded might not switch the implementation.
    // Tests below might implicitly rely on the *initial* config state when the test suite loaded.
    // For robust testing of mode switching, jest.resetModules() between tests or test suites
    // targeting different modes would be necessary.
     // Reset modules to ensure dynamic imports are re-evaluated based on config.mockMode
     // This is crucial for reliably testing the mockMode switch.
    jest.resetModules();
     // Re-import necessary modules after reset, allowing mocks to apply correctly
     const updatedService = require('../src/services/service'); // Re-require service
     // Note: This might change how delegationService is referenced in tests if not handled carefully.
     // Let's keep the original import and rely on module mocking logic update instead.

    mockRedeemImplementation = jest.fn(); // Reset the shared mock fn
    // Reset config mock for each test if necessary
    jest.mock('../src/config', () => ({
      __esModule: true,
      default: { mockMode: false }, // Default to real mode for these tests
    }));
  })

  // Restore original config state after all tests
  afterAll(() => {
     config.mockMode = originalMockMode!
     jest.resetModules(); // Clean up module cache
  })

  async function initializeService(mockImplementation: jest.Mock) {
    currentMockImpl = mockImplementation;
    mockRedeemImplementation = currentMockImpl; // Point the module mock to the current test's mock
    // Dynamically import the service to re-evaluate with the new mock
    const module = await import('../src/services/service');
    delegationService = module.delegationService;
    // Allow time for async import within service.ts to resolve
    await new Promise(resolve => setTimeout(resolve, 50)); 
  }

  // Test suite for when running in REAL mode (mockMode = false)
  describe('Real Mode (mockMode: false)', () => {
    beforeEach(async () => {
      // For real mode, the actual redeemDelegation (which we've mocked at module level) will be used.
      // We set up a specific mock for it here for assertion.
      await initializeService(jest.fn().mockResolvedValue('0xRealTransactionHash'));
    });

    it('should redeem delegation successfully (snake_case request)', async () => {
      const mockRequest = {
        signature: Buffer.from('test-sig-snake'),
        merchant_address: 'merchant-addr-snake',
        token_contract_address: 'token-addr-snake',
        token_amount: 101,
        token_decimals: mockTokenDecimals, // Added
        chain_id: mockChainId,           // Added
        network_name: mockNetworkName      // Added
      };
      const call = { request: mockRequest } as any;
      const callback = jest.fn();

      await delegationService.redeemDelegation(call, callback);

      expect(currentMockImpl).toHaveBeenCalledWith(
        mockRequest.signature,
        mockRequest.merchant_address,
        mockRequest.token_contract_address,
        mockRequest.token_amount,
        mockRequest.token_decimals, // Expect
        mockRequest.chain_id,       // Expect
        mockRequest.network_name    // Expect
      );
      expect(callback).toHaveBeenCalledWith(null, {
        transaction_hash: '0xRealTransactionHash',
        transactionHash: '0xRealTransactionHash',
        success: true,
        error_message: "",
        errorMessage: ""
      });
    });

    it('should redeem delegation successfully (camelCase request)', async () => {
      const mockRequest = {
        signature: Buffer.from('test-sig-camel'),
        merchantAddress: 'merchant-addr-camel',
        tokenContractAddress: 'token-addr-camel',
        token_amount: 202, // assuming token_amount is always snake_case from proto
        token_decimals: mockTokenDecimals, // Added
        chain_id: mockChainId,           // Added
        network_name: mockNetworkName      // Added
      };
      const call = { request: mockRequest } as any;
      const callback = jest.fn();

      await delegationService.redeemDelegation(call, callback);

      expect(currentMockImpl).toHaveBeenCalledWith(
        mockRequest.signature,
        mockRequest.merchantAddress,
        mockRequest.tokenContractAddress,
        mockRequest.token_amount,
        mockRequest.token_decimals, // Expect
        mockRequest.chain_id,       // Expect
        mockRequest.network_name    // Expect
      );
      expect(callback).toHaveBeenCalledWith(null, {
        transaction_hash: '0xRealTransactionHash',
        transactionHash: '0xRealTransactionHash',
        success: true,
        error_message: "",
        errorMessage: ""
      });
    });

    it('should handle errors during redemption', async () => {
      const error = new Error('Redemption failed');
      await initializeService(jest.fn().mockRejectedValue(error)); // Setup mock to throw
      
      const mockRequest = {
        signature: Buffer.from('test-sig-error'),
        merchant_address: 'merchant-addr-error',
        token_contract_address: 'token-addr-error',
        token_amount: 303,
        token_decimals: mockTokenDecimals, // Added
        chain_id: mockChainId,           // Added
        network_name: mockNetworkName      // Added
      };
      const call = { request: mockRequest } as any;
      const callback = jest.fn();

      await delegationService.redeemDelegation(call, callback);

      expect(currentMockImpl).toHaveBeenCalledWith(
        mockRequest.signature,
        mockRequest.merchant_address,
        mockRequest.token_contract_address,
        mockRequest.token_amount,
        mockRequest.token_decimals, // Expect
        mockRequest.chain_id,       // Expect
        mockRequest.network_name    // Expect
      );
      expect(callback).toHaveBeenCalledWith(null, {
        transaction_hash: "",
        transactionHash: "",
        success: false,
        error_message: 'Redemption failed',
        errorMessage: 'Redemption failed'
      });
    });
    
    // Test for missing chain_id
    it('should reject if chain_id is missing', async () => {
        const mockRequest = {
            signature: Buffer.from('test-sig-no-chainid'),
            merchant_address: 'merchant-addr',
            token_contract_address: 'token-addr',
            token_amount: 707,
            token_decimals: mockTokenDecimals,
            // chain_id: mockChainId, // Missing
            network_name: mockNetworkName
        };
        const call = { request: mockRequest } as any;
        const callback = jest.fn();

        await delegationService.redeemDelegation(call, callback);

        expect(currentMockImpl).not.toHaveBeenCalled();
        expect(callback).toHaveBeenCalledWith(null, expect.objectContaining({
            success: false,
            error_message: 'Missing or invalid chain_id in request'
        }));
    });

    // Test for missing network_name
    it('should reject if network_name is missing', async () => {
        const mockRequest = {
            signature: Buffer.from('test-sig-no-networkname'),
            merchant_address: 'merchant-addr',
            token_contract_address: 'token-addr',
            token_amount: 808,
            token_decimals: mockTokenDecimals,
            chain_id: mockChainId,
            // network_name: mockNetworkName // Missing
        };
        const call = { request: mockRequest } as any;
        const callback = jest.fn();

        await delegationService.redeemDelegation(call, callback);

        expect(currentMockImpl).not.toHaveBeenCalled();
        expect(callback).toHaveBeenCalledWith(null, expect.objectContaining({
            success: false,
            error_message: 'Missing network_name in request'
        }));
    });

  });

  // Add similar describe block for MOCK_MODE if those tests also need updating
  // For now, focusing on the failing REAL mode tests
}) 