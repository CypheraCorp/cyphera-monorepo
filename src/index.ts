// Load the protobuf definition
const PROTO_PATH = path.join(__dirname, 'proto/delegation.proto')

const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
  keepCase: false,  // Use camelCase instead of snake_case for better compatibility
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true
}) 