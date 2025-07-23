import { NextResponse, NextRequest } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { requireAuth } from '@/lib/auth/guards/require-auth';
import type { UpdateProductTokenRequest } from '@/types/product';
import logger from '@/lib/core/logger/logger';

interface RouteParams {
  params: Promise<Record<string, string>>;
}

/**
 * GET /api/products/[productId]/networks/[networkId]/tokens/[tokenId]
 * Gets a specific token by ID for a product's network
 */
export async function GET(request: NextRequest, { params }: RouteParams) {
  try {
    await requireAuth();
    const { api, userContext } = await getAPIContextFromSession(request);
    const { productId, networkId, tokenId } = await params;
    const token = await api.products.getProductTokenById(
      userContext,
      productId,
      networkId,
      tokenId
    );
    return NextResponse.json(token);
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
    }
    logger.error('Error getting token', { error });
    return NextResponse.json({ error: 'Failed to get token' }, { status: 500 });
  }
}

/**
 * PUT /api/products/[productId]/networks/[networkId]/tokens/[tokenId]
 * Updates a specific token by ID for a product's network
 */
export async function PUT(request: NextRequest, { params }: RouteParams) {
  try {
    await requireAuth();
    const body = (await request.json()) as UpdateProductTokenRequest;

    const { api, userContext } = await getAPIContextFromSession(request);
    const { productId, networkId, tokenId } = await params;
    const token = await api.products.updateProductToken(
      userContext,
      productId,
      networkId,
      tokenId,
      body
    );

    return NextResponse.json(token);
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
    }
    logger.error('Error updating token', { error });
    return NextResponse.json({ error: 'Failed to update token' }, { status: 500 });
  }
}

/**
 * DELETE /api/products/[productId]/networks/[networkId]/tokens/[tokenId]
 * Deletes a specific token by ID for a product's network
 */
export async function DELETE(request: NextRequest, { params }: RouteParams) {
  try {
    await requireAuth();
    const { api, userContext } = await getAPIContextFromSession(request);
    const { productId, networkId, tokenId } = await params;
    await api.products.deleteProductToken(userContext, productId, networkId, tokenId);

    // Return 204 No Content for successful deletion
    return new NextResponse(null, { status: 204 });
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
    }
    logger.error('Error deleting token', { error });
    return NextResponse.json({ error: 'Failed to delete token' }, { status: 500 });
  }
}
