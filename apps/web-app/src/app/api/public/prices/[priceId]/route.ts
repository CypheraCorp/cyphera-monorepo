import { NextRequest, NextResponse } from 'next/server';
import { PublicAPI } from '@/services/cyphera-api/public';
import logger from '@/lib/core/logger/logger';

// This route handles product fetching using the priceId route parameter for URL structure
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ priceId: string }> }
) {
  try {
    const { priceId: productId } = await params;

    const publicAPI = new PublicAPI();
    const product = await publicAPI.getPublicProduct(productId);

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
