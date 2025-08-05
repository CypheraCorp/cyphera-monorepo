import { NextRequest, NextResponse } from 'next/server';
import { SubscribeAPI } from '@/services/cyphera-api/subscribe';
import logger from '@/lib/core/logger/logger';
import { validateBody } from '@/lib/validation/validate';
import { subscribeRequestSchema } from '@/lib/validation/schemas/subscription';

interface RouteParams {
  params: Promise<{ productId: string }>;
}

/**
 * POST /api/pay/:productId/subscribe
 * Public Cyphera API endpoint to handle product subscriptions.
 * Uses the API Key for authentication.
 * Note: Public endpoints don't need CSRF protection
 */
export async function POST(
  request: NextRequest,
  { params }: RouteParams
) {
  try {
    // Get the productId from params
    const { productId } = await params;
    
    // Validate request body
    const { data: body, error: validationError } = await validateBody(request, subscribeRequestSchema);
    if (validationError) return validationError;
    
    // Log the incoming request for debugging
    logger.info('Subscribe endpoint called', {
      productId,
      body: JSON.stringify(body, null, 2),
      headers: Object.fromEntries(request.headers.entries())
    });
    
    if (!productId) {
      return NextResponse.json({ error: 'Product ID is required' }, { status: 400 });
    }

    // Merge product_id from params into body
    const requestData = {
      ...body!,
      product_id: productId
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
      requestData.product_id,
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