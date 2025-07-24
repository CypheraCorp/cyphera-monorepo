import { NextRequest, NextResponse } from 'next/server';
import { SubscribeAPI } from '@/services/cyphera-api/subscribe';
import logger from '@/lib/core/logger/logger';
import { validateBody } from '@/lib/validation/validate';
import { subscribeRequestSchema } from '@/lib/validation/schemas/subscription';

interface RouteParams {
  params: Promise<{ priceId: string }>;
}

/**
 * POST /api/public/prices/:priceId/subscribe
 * Public Cyphera API endpoint to handle price subscriptions.
 * Uses the API Key for authentication.
 * Note: Public endpoints don't need CSRF protection
 */
export async function POST(
  request: NextRequest,
  { params }: RouteParams
) {
  try {
    // Get the priceId from params
    const { priceId } = await params;
    
    // Validate request body
    const { data: body, error: validationError } = await validateBody(request, subscribeRequestSchema);
    if (validationError) return validationError;
    
    // Log the incoming request for debugging
    logger.info('Subscribe endpoint called', {
      priceId,
      body: JSON.stringify(body, null, 2),
      headers: Object.fromEntries(request.headers.entries())
    });
    
    if (!priceId) {
      return NextResponse.json({ error: 'Price ID is required' }, { status: 400 });
    }

    // Merge price_id from params into body
    const requestData = {
      ...body!,
      price_id: priceId
    };

    // Custom serializer for BigInt in delegation
    const replaceBigInt = (key: string, value: unknown): string | unknown => {
      if (typeof value === 'bigint') {
        return value.toString();
      }
      return value;
    };
    
    const processedDelegation = JSON.parse(JSON.stringify(requestData.delegation, replaceBigInt));

    // Initialize the API client
    const subscribeAPI = new SubscribeAPI();
    
    // Call the service method (which uses public headers internally)
    const subscriptionResult = await subscribeAPI.submitSubscription(
      requestData.price_id,
      requestData.product_token_id,
      requestData.token_amount,
      processedDelegation,
      requestData.subscriber_address
    );

    // Check if the service call was successful
    if (!subscriptionResult.success) {
      logger.error('Subscription failed', {
        message: subscriptionResult.message
      });
      
      return NextResponse.json(
        { success: false, message: subscriptionResult.message },
        { status: 400 }
      );
    }

    // Return the actual subscription data (not wrapped in service response)
    return NextResponse.json(subscriptionResult.data, { status: 200 });
  } catch (error) {
    // Log the full error
    logger.error('Error in subscribe endpoint', { 
      error,
      errorMessage: error instanceof Error ? error.message : 'Unknown error',
      errorStack: error instanceof Error ? error.stack : undefined
    });
    
    const message = error instanceof Error ? error.message : 'Failed to process subscription';
    return NextResponse.json({ success: false, message, error: error }, { status: 500 });
  }
}