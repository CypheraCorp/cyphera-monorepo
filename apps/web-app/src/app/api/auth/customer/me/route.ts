import { NextResponse } from 'next/server';
import { logger } from '@/lib/core/logger/logger';
import { UnifiedSessionService } from '@/lib/auth/session/unified-session';

/**
 * GET /api/auth/customer/me
 * Check if the current customer has a valid session
 */
export async function GET() {
  try {
    // Get customer session using unified service
    const session = await UnifiedSessionService.getByType('customer');

    if (!session) {
      return NextResponse.json({ error: 'No customer session found' }, { status: 401 });
    }

    // Return session data
    return NextResponse.json({
      customer: {
        customer_id: session.customer_id,
        customer_name: session.customer_name,
        customer_email: session.customer_email,
        wallet_address: session.wallet_address,
        wallet_id: session.wallet_id,
        finished_onboarding: session.finished_onboarding ?? false,
        created_at: session.created_at,
      },
      session,
    });
  } catch (error) {
    logger.error('Error checking customer session', {
      error: error instanceof Error ? error.message : error,
    });
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 });
  }
}
