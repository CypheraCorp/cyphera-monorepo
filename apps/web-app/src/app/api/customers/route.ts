import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import logger from '@/lib/core/logger/logger';

/**
 * GET /api/customers
 * Fetch customers with pagination
 */
export async function GET(request: NextRequest) {
  try {
    // Extract query parameters
    const { searchParams } = new URL(request.url);
    const page = Number(searchParams.get('page')) || 1;
    const limit = Number(searchParams.get('limit')) || 10;

    // Get authenticated API context from session
    const { api, userContext } = await getAPIContextFromSession(request);

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

/**
 * POST /api/customers
 * Create a new customer
 */
export async function POST(request: NextRequest) {
  try {
    // Parse request body
    const customerData = await request.json();

    // Get authenticated API context from session
    const { api, userContext } = await getAPIContextFromSession(request);

    // Create customer in backend
    const customer = await api.customers.createCustomer(userContext, customerData);

    return NextResponse.json(customer);
  } catch (error) {
    logger.error('Create customer API error', { error });
    const message = error instanceof Error ? error.message : 'Failed to create customer';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
