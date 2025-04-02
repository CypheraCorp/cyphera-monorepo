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

describe('delegationService', () => {
  let originalMockMode: boolean | undefined

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

  })

  // Restore original config state after all tests
  afterAll(() => {
     config.mockMode = originalMockMode!
     jest.resetModules(); // Clean up module cache
  })

  // --- Test Cases: Initial Call (Before Dynamic Import Completes) ---

  it.skip('should return "service not ready" if called immediately after require (real mode)', async () => {
    // Arrange
    config.mockMode = false;
    const { delegationService: currentService } = require('../src/services/service'); // Re-require
    const call = createMockCall({ signature: Buffer.from('sig') });
    const callback = createMockCallback();

    // Act: Call immediately
    await currentService.redeemDelegation(call, callback);

    // Assert: Expect "not ready" error, mock should NOT be called yet
    expect(callback).toHaveBeenCalledWith(null, {
      transaction_hash: "",
      transactionHash: "",
      success: false,
      error_message: "Service not ready yet, try again later",
      errorMessage: "Service not ready yet, try again later",
    });
    expect(mockRedeemImplementation).not.toHaveBeenCalled();
    expect(mockModeRedeemImplementation).not.toHaveBeenCalled();
  });

  it.skip('should return "service not ready" if called immediately after require (mock mode)', async () => {
    // Arrange
    config.mockMode = true;
    const { delegationService: currentService } = require('../src/services/service'); // Re-require
    const call = createMockCall({ signature: Buffer.from('sig') });
    const callback = createMockCallback();

    // Act: Call immediately
    await currentService.redeemDelegation(call, callback);

    // Assert: Expect "not ready" error, mock should NOT be called yet
    expect(callback).toHaveBeenCalledWith(null, {
      transaction_hash: "",
      transactionHash: "",
      success: false,
      error_message: "Service not ready yet, try again later",
      errorMessage: "Service not ready yet, try again later",
    });
    expect(mockRedeemImplementation).not.toHaveBeenCalled();
    expect(mockModeRedeemImplementation).not.toHaveBeenCalled();
  });

  // --- Test Cases: Subsequent Calls (After Dynamic Import Should Complete) ---

  const runAfterImport = async (setupFn: () => any, testFn: (service: any, mockImpl: jest.Mock) => Promise<void>) => {
      // Arrange: Set config and re-require
      setupFn();
      const { delegationService: currentService } = require('../src/services/service');

      // Allow time for the async import() and .then() in service.ts to resolve
      // Use setImmediate for potentially better yielding across I/O phases
      await new Promise(resolve => setImmediate(resolve));

      // Determine which mock should have been loaded
      const expectedMock = config.mockMode ? mockModeRedeemImplementation : mockRedeemImplementation;

      // Act & Assert within the provided test function
      await testFn(currentService, expectedMock);
  };

  it('should redeem delegation successfully (real mode, snake_case) after import', async () => {
    await runAfterImport(
        () => { config.mockMode = false; },
        async (currentService, currentMockImpl) => {
            // Arrange mock behavior AFTER import should be done
            currentMockImpl.mockResolvedValue('mock-tx-hash-snake');
            const mockRequest = {
                signature: Buffer.from('test-signature-snake'),
                merchant_address: 'merchant-addr-snake',
                token_contract_address: 'token-addr-snake',
                price: '101',
            };
            const call = createMockCall(mockRequest);
            const callback = createMockCallback();

            // Act
            await currentService.redeemDelegation(call, callback);

            // Assert
            expect(currentMockImpl).toHaveBeenCalledWith(
                mockRequest.signature,
                mockRequest.merchant_address,
                mockRequest.token_contract_address,
                mockRequest.price
            );
            expect(callback).toHaveBeenCalledWith(null, {
                transaction_hash: 'mock-tx-hash-snake',
                transactionHash: 'mock-tx-hash-snake',
                success: true,
                error_message: '',
                errorMessage: '',
            });
        }
    );
  });

  it('should redeem delegation successfully (real mode, camelCase) after import', async () => {
     await runAfterImport(
        () => { config.mockMode = false; },
        async (currentService, currentMockImpl) => {
            // Arrange
             currentMockImpl.mockResolvedValue('mock-tx-hash-camel');
            const mockRequest = {
                signature: Buffer.from('test-signature-camel'),
                merchantAddress: 'merchant-addr-camel',
                tokenContractAddress: 'token-addr-camel',
                price: '202',
            };
            const call = createMockCall(mockRequest);
            const callback = createMockCallback();

            // Act
            await currentService.redeemDelegation(call, callback);

            // Assert
            expect(currentMockImpl).toHaveBeenCalledWith(
                mockRequest.signature,
                mockRequest.merchantAddress,
                mockRequest.tokenContractAddress,
                mockRequest.price
            );
            expect(callback).toHaveBeenCalledWith(null, {
                transaction_hash: 'mock-tx-hash-camel',
                transactionHash: 'mock-tx-hash-camel',
                success: true,
                error_message: '',
                errorMessage: '',
            });
        }
    );
  });

  it('should handle errors during redemption (real mode) after import', async () => {
     await runAfterImport(
        () => { config.mockMode = false; },
        async (currentService, currentMockImpl) => {
            // Arrange
            const errorMessage = 'Blockchain transaction failed';
            currentMockImpl.mockRejectedValue(new Error(errorMessage));
            const mockRequest = {
                signature: Buffer.from('test-signature-error'),
                merchant_address: 'merchant-addr-err',
                token_contract_address: 'token-addr-err',
                price: '303',
            };
            const call = createMockCall(mockRequest);
            const callback = createMockCallback();

            // Act
            await currentService.redeemDelegation(call, callback);

            // Assert
            expect(currentMockImpl).toHaveBeenCalledWith(
                 mockRequest.signature,
                 mockRequest.merchant_address,
                 mockRequest.token_contract_address,
                 mockRequest.price
             );
            expect(callback).toHaveBeenCalledWith(null, {
                transaction_hash: '',
                transactionHash: '',
                success: false,
                error_message: errorMessage,
                errorMessage: errorMessage,
            });
        }
    );
  });

  it('should handle non-Error exceptions (real mode) after import', async () => {
     await runAfterImport(
        () => { config.mockMode = false; },
        async (currentService, currentMockImpl) => {
            // Arrange
            const errorString = 'Something weird happened';
            currentMockImpl.mockRejectedValue(errorString);
            const mockRequest = {
                signature: Buffer.from('test-signature-non-error'),
                merchant_address: 'merchant-addr-non-err',
                token_contract_address: 'token-addr-non-err',
                price: '404',
            };
            const call = createMockCall(mockRequest);
            const callback = createMockCallback();

            // Act
            await currentService.redeemDelegation(call, callback);

            // Assert
             expect(currentMockImpl).toHaveBeenCalledTimes(1);
             expect(callback).toHaveBeenCalledWith(null, {
                 transaction_hash: "",
                 transactionHash: "",
                 success: false,
                 error_message: errorString,
                 errorMessage: errorString,
             });
        }
     );
  });

  // Tests for missing parameters (after import)
 it('should attempt redemption (real mode, missing sig) after import', async () => {
    await runAfterImport(
        () => { config.mockMode = false; },
        async (currentService, currentMockImpl) => {
            currentMockImpl.mockResolvedValue('tx-hash-no-sig');
            const mockRequest = {
                merchant_address: 'merchant-addr-no-sig',
                token_contract_address: 'token-addr-no-sig',
                price: '606',
            };
            const call = createMockCall(mockRequest);
            const callback = createMockCallback();

            await currentService.redeemDelegation(call, callback);

            expect(currentMockImpl).toHaveBeenCalledWith(
                undefined,
                mockRequest.merchant_address,
                mockRequest.token_contract_address,
                mockRequest.price
            );
            expect(callback).toHaveBeenCalledWith(null, expect.objectContaining({ success: true, transactionHash: 'tx-hash-no-sig' }));
        }
    );
 });

 it('should attempt redemption (real mode, missing merchant) after import', async () => {
      await runAfterImport(
        () => { config.mockMode = false; },
        async (currentService, currentMockImpl) => {
            currentMockImpl.mockResolvedValue('tx-hash-no-merchant');
            const mockRequest = {
                signature: Buffer.from('sig-no-merchant'),
                token_contract_address: 'token-addr-no-merchant',
                price: '707',
            };
            const call = createMockCall(mockRequest);
            const callback = createMockCallback();

            await currentService.redeemDelegation(call, callback);

            expect(currentMockImpl).toHaveBeenCalledWith(
                mockRequest.signature,
                undefined,
                mockRequest.token_contract_address,
                mockRequest.price
            );
            expect(callback).toHaveBeenCalledWith(null, expect.objectContaining({ success: true, transactionHash: 'tx-hash-no-merchant' }));
        }
    );
 });

 it('should attempt redemption (real mode, missing token) after import', async () => {
      await runAfterImport(
        () => { config.mockMode = false; },
        async (currentService, currentMockImpl) => {
            currentMockImpl.mockResolvedValue('tx-hash-no-token');
            const mockRequest = {
                signature: Buffer.from('sig-no-token'),
                merchant_address: 'merchant-addr-no-token',
                price: '808',
            };
            const call = createMockCall(mockRequest);
            const callback = createMockCallback();

            await currentService.redeemDelegation(call, callback);

            expect(currentMockImpl).toHaveBeenCalledWith(
                mockRequest.signature,
                mockRequest.merchant_address,
                undefined,
                mockRequest.price
            );
            expect(callback).toHaveBeenCalledWith(null, expect.objectContaining({ success: true, transactionHash: 'tx-hash-no-token' }));
        }
    );
 });

 it('should attempt redemption (real mode, missing price) after import', async () => {
     await runAfterImport(
        () => { config.mockMode = false; },
        async (currentService, currentMockImpl) => {
            currentMockImpl.mockResolvedValue('tx-hash-no-price');
            const mockRequest = {
                signature: Buffer.from('sig-no-price'),
                merchant_address: 'merchant-addr-no-price',
                token_contract_address: 'token-addr-no-price',
            };
            const call = createMockCall(mockRequest);
            const callback = createMockCallback();

            await currentService.redeemDelegation(call, callback);

            expect(currentMockImpl).toHaveBeenCalledWith(
                mockRequest.signature,
                mockRequest.merchant_address,
                mockRequest.token_contract_address,
                undefined
            );
            expect(callback).toHaveBeenCalledWith(null, expect.objectContaining({ success: true, transactionHash: 'tx-hash-no-price' }));
        }
    );
 });

}) 