import { NextResponse } from 'next/server';
import { TokenQuotePayload } from '@/types/token';
import { TokensAPI } from '@/services/cyphera-api/tokens';
import logger from '@/lib/core/logger/logger';
/**
 * GET /api/tokens/quote
 * Gets the price of a token in USD
 */
export async function POST(request: Request) {
  try {
    const tokensAPI = new TokensAPI();
    const payload = await request.json();

    if (!payload.token_symbol || !payload.fiat_symbol) {
      return NextResponse.json({ error: 'Missing required parameters' }, { status: 400 });
    }

    // TokenQuotePayload
    const tokenQuotePayload: TokenQuotePayload = {
      token_symbol: payload.token_symbol,
      fiat_symbol: payload.fiat_symbol,
    };

    const result = await tokensAPI.getTokenQuote(tokenQuotePayload);

    // Return response with no-cache headers
    const response = NextResponse.json(result);
    response.headers.set('Cache-Control', 'no-store, no-cache, must-revalidate');
    response.headers.set('Pragma', 'no-cache');
    response.headers.set('Expires', '0');
    return response;
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
    }
    logger.error('Error getting token price', { error });
    const message = error instanceof Error ? error.message : 'Failed to get token price';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
