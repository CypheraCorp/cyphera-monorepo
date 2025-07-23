// Mock the logger to prevent console output during tests

// Save the original console methods
const originalConsole = {
  log: console.log,
  info: console.info,
  warn: console.warn,
  error: console.error,
  debug: console.debug
};

// Replace with mock implementations
console.log = jest.fn();
console.info = jest.fn();
console.warn = jest.fn();
console.error = jest.fn();
console.debug = jest.fn();

// Since this is a setup file, don't use afterAll/beforeAll
// The console mocking will persist throughout the test session 