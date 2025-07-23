// Bundle analyzer configuration
// eslint-disable-next-line @typescript-eslint/no-require-imports
const withBundleAnalyzer = require('@next/bundle-analyzer')({
  enabled: process.env.ANALYZE === 'true',
});

/** @type {import('next').NextConfig} */
const nextConfig = {
  images: {
    // Enable image optimization for better performance
    unoptimized: false,
    // Support modern image formats
    formats: ['image/avif', 'image/webp'],
    // Set cache TTL for images (1 year)
    minimumCacheTTL: 60 * 60 * 24 * 365,
    // Optimize image sizes
    deviceSizes: [640, 750, 828, 1080, 1200, 1920, 2048, 3840],
    imageSizes: [16, 32, 48, 64, 96, 128, 256, 384],
    // Configure remote image domains for profile images
    remotePatterns: [
      {
        protocol: 'https',
        hostname: 'lh3.googleusercontent.com',
        pathname: '/a/**',
      },
      {
        protocol: 'https',
        hostname: 'avatars.githubusercontent.com',
        pathname: '/**',
      },
      {
        protocol: 'https',
        hostname: 'cdn.auth0.com',
        pathname: '/**',
      },
      {
        protocol: 'https',
        hostname: 'graph.facebook.com',
        pathname: '/**',
      },
      {
        protocol: 'https',
        hostname: 'pbs.twimg.com',
        pathname: '/**',
      },
    ],
  },
  typescript: {
    ignoreBuildErrors: false,
  },
  eslint: {
    ignoreDuringBuilds: true,
  },

  // Performance optimizations
  compiler: {
    // Remove console logs in production except warnings and errors
    removeConsole: process.env.NODE_ENV === 'production' ? {
      exclude: ['error', 'warn'],
    } : false,
  },

  // Optimize bundle splitting
  experimental: {
    optimizePackageImports: [
      'lucide-react',
      '@radix-ui/react-icons',
      'date-fns',
      '@web3auth/modal',
      '@web3auth/base',
      '@web3auth/ethereum-provider',
      '@web3auth/account-abstraction-provider',
      '@web3auth/metamask-adapter',
      '@web3auth/wallet-connect-v2-adapter',
      '@tanstack/react-query',
      'react-hook-form',
      'zod',
      'framer-motion',
      'wagmi',
      'viem',
    ],
  },

  // Add headers to fix Web3Auth popup issues and improve caching
  async headers() {
    return [
      {
        source: '/(.*)',
        headers: [
          {
            key: 'Cross-Origin-Opener-Policy',
            value: 'same-origin-allow-popups',
          },
          {
            key: 'Cross-Origin-Embedder-Policy',
            value: 'unsafe-none',
          },
        ],
      },
      // Cache static assets aggressively
      {
        source: '/images/:path*',
        headers: [
          {
            key: 'Cache-Control',
            value: 'public, max-age=31536000, immutable',
          },
        ],
      },
      // Cache icons specifically
      {
        source: '/(favicon.ico|icon.ico|icon.png|icon.svg)',
        headers: [
          {
            key: 'Cache-Control',
            value: 'public, max-age=31536000, immutable',
          },
        ],
      },
    ];
  },

  // Webpack config optimized for performance
  webpack: (config, { isServer }) => {
    // EXCLUDE web3auth-examples from bundling
    config.resolve.alias = {
      ...config.resolve.alias,
      // Exclude the entire web3auth-examples directory
      'web3auth-examples': false,
      '@react-native-async-storage/async-storage': false,
    };

    // Exclude web3auth-examples directory from module resolution
    config.externals = config.externals || [];
    config.externals.push(/^web3auth-examples/);

    // Optimize module resolution
    config.resolve.modules = ['node_modules'];

    // Fix exports issue by ensuring proper module format
    config.module.rules.push({
      test: /\.m?js$/,
      type: 'javascript/auto',
      resolve: {
        fullySpecified: false,
      },
    });

    if (!isServer) {
      // Simplified fallback configuration
      config.resolve.fallback = {
        ...config.resolve.fallback,
        process: 'process/browser',
        // Ignore React Native dependencies that don't work in browser
        '@react-native-async-storage/async-storage': false,
      };

      // Optimize client-side bundle
      config.optimization = {
        ...config.optimization,
        splitChunks: {
          chunks: 'all',
          cacheGroups: {
            // Core React and Next.js
            framework: {
              test: /[\\/]node_modules[\\/](react|react-dom|next)[\\/]/,
              name: 'framework',
              chunks: 'all',
              priority: 40,
              enforce: true,
            },
            // CSS files - keep them together to avoid syntax errors
            styles: {
              test: /\.css$/,
              name: 'styles',
              chunks: 'all',
              priority: 35,
              enforce: true,
            },
            // Web3Auth packages
            web3auth: {
              test: /[\\/]node_modules[\\/]@web3auth[\\/]/,
              name: 'web3auth',
              chunks: 'all',
              priority: 30,
              enforce: true,
            },
            // Blockchain packages
            blockchain: {
              test: /[\\/]node_modules[\\/](wagmi|viem|permissionless)[\\/]/,
              name: 'blockchain',
              chunks: 'all',
              priority: 25,
              enforce: true,
            },
            // UI components
            radix: {
              test: /[\\/]node_modules[\\/]@radix-ui[\\/]/,
              name: 'radix-ui',
              chunks: 'all',
              priority: 20,
              enforce: true,
            },
            // Form and validation
            forms: {
              test: /[\\/]node_modules[\\/](react-hook-form|zod|@hookform)[\\/]/,
              name: 'forms',
              chunks: 'all',
              priority: 18,
              enforce: true,
            },
            // Animation libraries
            animations: {
              test: /[\\/]node_modules[\\/](framer-motion|lucide-react)[\\/]/,
              name: 'animations',
              chunks: 'all',
              priority: 15,
              enforce: true,
            },
            // Other vendor packages
            vendor: {
              test: /[\\/]node_modules[\\/]/,
              name: 'vendors',
              chunks: 'all',
              priority: 10,
              enforce: true,
            },
          },
        },
      };
    }

    return config;
  },
};

module.exports = withBundleAnalyzer(nextConfig);
