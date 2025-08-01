import { NextRequest, NextResponse } from 'next/server';
import { PublicAPI } from '@/services/cyphera-api/public';
import logger from '@/lib/core/logger/logger';

// This route handles product fetching using the productId route parameter
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ productId: string }> }
) {
  try {
    const { productId } = await params;

    const publicAPI = new PublicAPI();
    // Call the backend API directly to avoid circular reference
    const product = await publicAPI.getPublicProductById(productId);

    // Return response with no-cache headers
    const response = NextResponse.json(product);
    response.headers.set('Cache-Control', 'no-store, no-cache, must-revalidate');
    response.headers.set('Pragma', 'no-cache');
    response.headers.set('Expires', '0');
    return response;
  } catch (error) {
    logger.error('Failed to fetch product', { error });
    return NextResponse.json({ error: 'Failed to fetch product' }, { status: 500 });
  }
}