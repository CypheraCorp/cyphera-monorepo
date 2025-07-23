import { NextResponse } from 'next/server';
import logger from '@/lib/core/logger/logger';

/**
 * GET handler for fetching the Cyphera delegate address
 */
export async function GET() {
  try {
    // Get the delegate address from environment variables
    const delegateAddress = process.env.CYPHERA_DELEGATE_ADDRESS;

    if (!delegateAddress || !delegateAddress.startsWith('0x')) {
      logger.error('CYPHERA_DELEGATE_ADDRESS is not configured or is invalid');
      return NextResponse.json(
        {
          success: false,
          message:
            'Cyphera delegate address is not configured correctly in the server. Please contact support.',
          address: null,
        },
        { status: 500 }
      );
    }

    // Return the delegate address
    return NextResponse.json({
      success: true,
      message: 'Delegate address retrieved successfully',
      address: delegateAddress,
    });
  } catch (error) {
    logger.error('Error retrieving delegate address', { error });
    return NextResponse.json(
      {
        success: false,
        message: error instanceof Error ? error.message : 'An unexpected error occurred',
        address: null,
      },
      { status: 500 }
    );
  }
}
