import { NextRequest, NextResponse } from 'next/server';
import logger from '@/lib/core/logger/logger';

// Supported blockchain values (update this list based on current Circle support if needed)
const SUPPORTED_BLOCKCHAINS = [
  'MATIC-AMOY',
  'ETH-SEPOLIA',
  'AVAX-FUJI',
  'ARB-SEPOLIA',
  'BASE-SEPOLIA',
  'UNI-SEPOLIA',
  'OP-SEPOLIA',
  'SOL-DEVNET',
];

export async function POST(request: NextRequest) {
  try {
    // Parse the request body
    const body = await request.json();
    const { address, blockchain, usdc = true } = body;

    // Validate required fields and types
    if (!address || typeof address !== 'string' || address.trim() === '') {
      return NextResponse.json(
        { success: false, message: 'Address is required and must be a non-empty string' },
        { status: 400 }
      );
    }

    if (!blockchain || typeof blockchain !== 'string' || blockchain.trim() === '') {
      return NextResponse.json(
        { success: false, message: 'Blockchain is required and must be a non-empty string' },
        { status: 400 }
      );
    }

    // Validate blockchain value against supported list
    if (!SUPPORTED_BLOCKCHAINS.includes(blockchain)) {
      return NextResponse.json(
        {
          success: false,
          message: `Unsupported blockchain: ${blockchain}. Supported values are: ${SUPPORTED_BLOCKCHAINS.join(', ')}`,
        },
        { status: 400 }
      );
    }

    // Validate usdc type if provided explicitly (it defaults to true otherwise)
    if (body.hasOwnProperty('usdc') && typeof usdc !== 'boolean') {
      return NextResponse.json(
        { success: false, message: 'Optional field "usdc" must be a boolean' },
        { status: 400 }
      );
    }

    // Get Circle API key from environment variables
    const apiKey = process.env.CIRCLE_API_KEY;
    if (!apiKey) {
      logger.error('CIRCLE_API_KEY is not defined in environment variables');
      return NextResponse.json(
        { success: false, message: 'Server configuration error' },
        { status: 500 }
      );
    }

    const response = await fetch('https://api.circle.com/v1/faucet/drips', {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${apiKey}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        usdc,
        blockchain,
        address,
      }),
    });

    // Handle different response types
    if (response.ok) {
      let responseData;

      // Check if there's a response body
      const responseText = await response.text();
      if (responseText) {
        try {
          responseData = JSON.parse(responseText);
        } catch {
          // If response is not JSON but status is OK, it's still a success
        }
      }

      return NextResponse.json({
        success: true,
        message: 'Tokens requested successfully',
        data: responseData,
      });
    } else {
      // Handle error responses
      let errorMessage = 'Failed to request tokens from Circle API';
      let errorData;

      try {
        const errorResponse = await response.text();
        if (errorResponse) {
          try {
            errorData = JSON.parse(errorResponse);
            errorMessage = errorData.message || errorMessage;
          } catch (error) {
            logger.error('Error parsing error response', { error });
          }
        }
      } catch (error) {
        logger.error('Error reading error response', { error });
      }

      logger.error('Circle API error', { status: response.status, errorMessage, errorData });

      return NextResponse.json(
        {
          success: false,
          message: errorMessage,
          data: errorData,
        },
        { status: response.status }
      );
    }
  } catch (error) {
    logger.error('Unexpected error in faucet API', { error });
    return NextResponse.json(
      {
        success: false,
        message: error instanceof Error ? error.message : 'An unexpected error occurred',
      },
      { status: 500 }
    );
  }
}
