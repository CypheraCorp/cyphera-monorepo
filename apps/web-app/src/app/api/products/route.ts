import { NextRequest, NextResponse } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { requireAuth } from '@/lib/auth/guards/require-auth';
import type { CreateProductRequest } from '@/types/product';
import { logger } from '@/lib/core/logger/logger';

/**
 * GET /api/products
 * Gets all products for the current account
 */
export async function GET(request: NextRequest) {
  try {
    await requireAuth();

    // Handle potential pagination params from request URL
    const { searchParams } = new URL(request.url);
    const page = searchParams.get('page') || undefined;
    const limit = searchParams.get('limit') || undefined;

    const { api, userContext } = await getAPIContextFromSession(request);

    // Fetch fresh data without caching
    const productsResponse = await api.products.getProducts(userContext, {
      page: page ? Number(page) : undefined,
      limit: limit ? Number(limit) : undefined,
    });

    // Return response with no-cache headers
    const response = NextResponse.json(productsResponse);
    response.headers.set('Cache-Control', 'no-store, no-cache, must-revalidate');
    response.headers.set('Pragma', 'no-cache');
    response.headers.set('Expires', '0');
    return response;
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
    }
    logger.error('Error getting products', {
      error: error instanceof Error ? error.message : error,
    });
    const message = error instanceof Error ? error.message : 'Failed to get products';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}

/**
 * POST /api/products
 * Creates a new product
 */
export async function POST(request: NextRequest) {
  try {
    await requireAuth();
    const body = (await request.json()) as CreateProductRequest;

    const { api, userContext } = await getAPIContextFromSession(request);
    // Pass context to API call
    const product = await api.products.createProduct(userContext, body);

    return NextResponse.json(product);
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
    }
    logger.error('Error creating product', {
      error: error instanceof Error ? error.message : error,
    });
    const message = error instanceof Error ? error.message : 'Failed to create product';
    // Check if it might be a validation error (e.g., 400, 422) vs server error (500)
    // This requires the API to return appropriate status codes
    interface APIError extends Error {
      status?: number;
    }
    const apiError = error as APIError;
    const status = apiError?.status === 400 || apiError?.status === 422 ? 400 : 500;
    return NextResponse.json({ error: message }, { status });
  }
}
