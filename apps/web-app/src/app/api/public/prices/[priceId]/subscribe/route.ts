import { NextRequest, NextResponse } from 'next/server';
import { SubscribeAPI } from '@/services/cyphera-api/subscribe';
import logger from '@/lib/core/logger/logger';

interface RouteParams {
  params: Promise<{
    priceId: string;
  }>;
}

/**
 * POST /api/public/prices/:priceId/subscribe
 * Public Cyphera API endpoint to handle price subscriptions.
 * Uses the API Key for authentication.
 */
export async function POST(request: NextRequest, { params }: RouteParams) {
  try {
    const subscribeAPI = new SubscribeAPI();
    const { priceId } = await params; // Await params in Next.js 15
    if (!priceId) {
      return NextResponse.json({ success: false, message: 'Missing price ID' }, { status: 400 });
    }

    let body;
    try {
      body = await request.json();
    } catch (error) {
      logger.error('Invalid JSON in request body', { error });
      return NextResponse.json(
        { success: false, message: 'Invalid request body' },
        { status: 400 }
      );
    }

    const { subscriber_address, product_token_id, delegation, token_amount } = body;

    // Keep validation
    if (!subscriber_address) {
      return NextResponse.json(
        { success: false, message: 'Missing subscriber_address' },
        { status: 400 }
      );
    }
    if (!product_token_id) {
      return NextResponse.json(
        { success: false, message: 'Missing product_token_id' },
        { status: 400 }
      );
    }
    if (!delegation) {
      return NextResponse.json({ success: false, message: 'Missing delegation' }, { status: 400 });
    }
    if (!token_amount || typeof token_amount !== 'string') {
      return NextResponse.json(
        { success: false, message: 'Invalid token amount' },
        { status: 400 }
      );
    }

    // Custom serializer for BigInt
    const replaceBigInt = (key: string, value: unknown): string | unknown => {
      if (typeof value === 'bigint') {
        return value.toString();
      }
      return value;
    };
    const processedDelegation = JSON.parse(JSON.stringify(delegation, replaceBigInt));

    // Call the service method (which uses public headers internally)
    const subscriptionResult = await subscribeAPI.submitSubscription(
      priceId,
      product_token_id,
      token_amount,
      processedDelegation,
      subscriber_address
    );

    // Check if the service call was successful
    if (!subscriptionResult.success) {
      return NextResponse.json(
        { success: false, message: subscriptionResult.message },
        { status: 400 }
      );
    }

    // Return the actual subscription data (not wrapped in service response)
    return NextResponse.json(subscriptionResult.data, { status: 200 });
  } catch (error) {
    // Catch errors from getAPIContext or unexpected issues
    logger.error('Error in subscribe endpoint', { error });
    const message = error instanceof Error ? error.message : 'Failed to process subscription';
    return NextResponse.json({ success: false, message }, { status: 500 });
  }
}
