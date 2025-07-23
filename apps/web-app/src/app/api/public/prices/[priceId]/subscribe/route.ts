import { NextRequest, NextResponse } from 'next/server';
import { SubscribeAPI } from '@/services/cyphera-api/subscribe';
import logger from '@/lib/core/logger/logger';
import { withValidation } from '@/lib/validation/validate';
import { subscribeRequestSchema } from '@/lib/validation/schemas/subscription';
import { priceIdParamSchema } from '@/lib/validation/schemas/product';
import { z } from 'zod';

interface RouteParams {
  params: Promise<Record<string, string>>;
}

/**
 * POST /api/public/prices/:priceId/subscribe
 * Public Cyphera API endpoint to handle price subscriptions.
 * Uses the API Key for authentication.
 * Note: Public endpoints don't need CSRF protection
 */
export const POST = withValidation(
  { 
    bodySchema: subscribeRequestSchema.extend({
      price_id: z.string().uuid('Invalid price ID format')
    }),
    paramsSchema: priceIdParamSchema 
  },
  async (request, { body, params }) => {
    try {
      const subscribeAPI = new SubscribeAPI();
      
      // Ensure we have the priceId from params
      if (!params?.priceId) {
        return NextResponse.json({ error: 'Price ID is required' }, { status: 400 });
      }

      // Merge price_id from params into body for validation
      const requestData = {
        ...body,
        price_id: params.priceId
      };

      // Custom serializer for BigInt in delegation
      const replaceBigInt = (key: string, value: unknown): string | unknown => {
        if (typeof value === 'bigint') {
          return value.toString();
        }
        return value;
      };
      const processedDelegation = JSON.parse(JSON.stringify(requestData.delegation, replaceBigInt));

      // Since validation passed, all fields are guaranteed to be present
      // Call the service method (which uses public headers internally)
      const subscriptionResult = await subscribeAPI.submitSubscription(
        requestData.price_id!,
        requestData.product_token_id!,
        requestData.token_amount!,
        processedDelegation,
        requestData.subscriber_address!
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
);
