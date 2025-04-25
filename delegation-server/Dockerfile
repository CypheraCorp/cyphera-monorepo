# Use an official Node.js runtime as a parent image
# Choose a version compatible with your code (e.g., 18) and consider Alpine for smaller images
FROM node:18-alpine AS base

# Set the working directory in the container
WORKDIR /usr/src/app

# Install dependencies first to leverage Docker cache
# Copy package.json and lock file
COPY package.json package-lock.json* ./

# Use npm ci for clean, reproducible installs based on lockfile
# Only install production dependencies - crucial for smaller final image
RUN npm ci --omit=dev

# Copy the rest of your application code
# Ensure build artifacts (dist folder) are included if needed by the CMD
COPY . .

# If your build step (`npm run build`) needs to be run inside the container
# uncomment the following line. This is often needed if the build generates
# JS files from TS that are needed by the CMD step.
# RUN npm run build

# Expose the port the gRPC server listens on (Verify this port in your code)
EXPOSE 50051

# Define the command to run your application
# Use 'dist/index.js' assuming your build step outputs here.
# Adjust if your entry point is different (e.g., server.js, dist/server.js).
CMD ["node", "dist/index.js"] 