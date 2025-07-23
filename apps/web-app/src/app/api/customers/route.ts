import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import logger from '@/lib/core/logger/logger';
import { withCSRFProtection } from '@/lib/security/csrf-middleware';
import { withValidation } from '@/lib/validation/validate';
import { createCustomerSchema, customerQuerySchema } from '@/lib/validation/schemas/customer';

/**
 * GET /api/customers
 * Fetch customers with pagination
 */
export const GET = withValidation(
  { querySchema: customerQuerySchema },
  async (request, { query }) => {
    try {
      // Get authenticated API context from session
      const { api, userContext } = await getAPIContextFromSession(request);

      // Use validated query params
      const page = query?.page || 1;
      const limit = query?.limit || 10;

      // Fetch fresh data without caching
      const customers = await api.customers.getCustomers(userContext, { page, limit });

      // Return response with no-cache headers
      const response = NextResponse.json(customers);
      response.headers.set('Cache-Control', 'no-store, no-cache, must-revalidate');
      response.headers.set('Pragma', 'no-cache');
      response.headers.set('Expires', '0');
      return response;
    } catch (error) {
      logger.error('Customers API error', { error });
      const message = error instanceof Error ? error.message : 'Failed to fetch customers';
      return NextResponse.json({ error: message }, { status: 500 });
    }
  }
);

/**
 * POST /api/customers
 * Create a new customer
 */
export const POST = withCSRFProtection(
  withValidation(
    { bodySchema: createCustomerSchema },
    async (request, { body }) => {
      try {
        // Get authenticated API context from session
        const { api, userContext } = await getAPIContextFromSession(request);

        if (!body) {
          return NextResponse.json({ error: 'Request body is required' }, { status: 400 });
        }

        // Create customer with validated data
        const customer = await api.customers.createCustomer(userContext, body);

        return NextResponse.json(customer);
      } catch (error) {
        logger.error('Create customer API error', { error });
        const message = error instanceof Error ? error.message : 'Failed to create customer';
        return NextResponse.json({ error: message }, { status: 500 });
      }
    }
  )
);
