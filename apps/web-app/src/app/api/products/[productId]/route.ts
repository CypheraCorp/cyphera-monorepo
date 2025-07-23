import { NextResponse, NextRequest } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { requireAuth } from '@/lib/auth/guards/require-auth';
import type { UpdateProductRequest } from '@/types/product';
import logger from '@/lib/core/logger/logger';

interface RouteParams {
  params: Promise<{
    productId: string;
  }>;
}

/**
 * GET /api/products/[productId]
 * Gets a specific product by ID
 */
export async function GET(request: NextRequest, { params }: RouteParams) {
  try {
    await requireAuth();
    const { api, userContext } = await getAPIContextFromSession(request);
    const { productId } = await params;
    const product = await api.products.getProductById(userContext, productId);
    return NextResponse.json(product);
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
    }
    logger.error('Error getting product', { error });
    const message = error instanceof Error ? error.message : 'Failed to get product';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}

/**
 * PUT /api/products/[productId]
 * Updates a specific product by ID
 */
export async function PUT(request: NextRequest, { params }: RouteParams) {
  try {
    await requireAuth();
    const body = (await request.json()) as UpdateProductRequest;

    const { api, userContext } = await getAPIContextFromSession(request);
    const { productId } = await params;
    const product = await api.products.updateProduct(userContext, productId, body);

    return NextResponse.json(product);
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
    }
    logger.error('Error updating product', { error });
    const message = error instanceof Error ? error.message : 'Failed to update product';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}

/**
 * DELETE /api/products/[productId]
 * Deletes a specific product by ID
 */
export async function DELETE(request: NextRequest, { params }: RouteParams) {
  try {
    await requireAuth();
    const { api, userContext } = await getAPIContextFromSession(request);
    const { productId } = await params;
    await api.products.deleteProduct(userContext, productId);

    return new NextResponse(null, { status: 204 });
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
    }
    logger.error('Error deleting product', { error });
    const message = error instanceof Error ? error.message : 'Failed to delete product';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
