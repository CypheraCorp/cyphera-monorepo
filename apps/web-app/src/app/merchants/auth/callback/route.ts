import { NextRequest, NextResponse } from 'next/server';

/**
 * Auth callback route - redirects to main page for Web3Auth handling
 * This is used by Web3Auth for OAuth redirects
 */
export async function GET(request: NextRequest) {
  // Web3Auth will handle the actual authentication on the client side
  // We just need to redirect to the main page
  return NextResponse.redirect(new URL('/', request.url));
}
