import { NextRequest, NextResponse } from 'next/server';
import { csrf } from '@/lib/security/csrf';

/**
 * GET /api/auth/csrf
 * Returns a CSRF token for the client to use in subsequent requests
 */
export async function GET(request: NextRequest) {
  try {
    // Generate CSRF token
    const token = csrf.create(request);
    
    // Set CSRF cookie
    const response = NextResponse.json({ token });
    
    // The csrf library will handle setting the cookie
    return response;
  } catch (error) {
    console.error('Failed to generate CSRF token:', error);
    return NextResponse.json(
      { error: 'Failed to generate CSRF token' },
      { status: 500 }
    );
  }
}