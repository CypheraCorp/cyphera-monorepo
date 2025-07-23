import { NextResponse, NextRequest } from 'next/server';
import { getAPIContextFromSession } from '@/lib/api/server/server-api';
import { requireAuth } from '@/lib/auth/guards/require-auth';
import type { UpdateProductRequest } from '@/types/product';
import logger from '@/lib/core/logger/logger';
import { withCSRFProtection } from '@/lib/security/csrf-middleware';
import { withValidation } from '@/lib/validation/validate';
import { updateProductSchema, productIdParamSchema } from '@/lib/validation/schemas/product';

interface RouteParams {
  params: Promise<Record<string, string>>;
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
export const PUT = withCSRFProtection(
  async (request: NextRequest, context: RouteParams) => {
    try {
      await requireAuth();
      
      // Validate params
      const { productId } = await context.params;
      const paramsValidation = productIdParamSchema.safeParse({ productId });
      if (!paramsValidation.success) {
        return NextResponse.json(
          { error: 'Invalid product ID format' },
          { status: 400 }
        );
      }
      
      // Validate body
      const body = await request.json();
      const bodyValidation = updateProductSchema.safeParse(body);
      if (!bodyValidation.success) {
        return NextResponse.json(
          { 
            error: 'Validation failed',
            details: bodyValidation.error.errors 
          },
          { status: 400 }
        );
      }

      const { api, userContext } = await getAPIContextFromSession(request);
      const product = await api.products.updateProduct(userContext, productId, bodyValidation.data);

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
);

/**
 * DELETE /api/products/[productId]
 * Deletes a specific product by ID
 */
export const DELETE = withCSRFProtection(
  async (request: NextRequest, context: RouteParams) => {
    try {
      await requireAuth();
      
      // Validate params
      const { productId } = await context.params;
      const paramsValidation = productIdParamSchema.safeParse({ productId });
      if (!paramsValidation.success) {
        return NextResponse.json(
          { error: 'Invalid product ID format' },
          { status: 400 }
        );
      }
      
      const { api, userContext } = await getAPIContextFromSession(request);
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
);
