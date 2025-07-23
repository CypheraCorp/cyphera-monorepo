import { NextRequest, NextResponse } from 'next/server';
import { logger } from '@/lib/core/logger/logger';

/**
 * GET /api/debug/session
 * Debug endpoint to check session status
 */
export async function GET(request: NextRequest) {
  try {
    // Get session from cookie
    const sessionCookie = request.cookies.get('cyphera-session')?.value;
    const customerSessionCookie = request.cookies.get('cyphera-customer-session')?.value;

    // Decode session data if available
    let sessionData = null;
    let customerSessionData = null;

    if (sessionCookie) {
      try {
        const decodedSession = Buffer.from(sessionCookie, 'base64').toString('utf-8');
        sessionData = JSON.parse(decodedSession);
      } catch (e) {
        logger.debug('Failed to decode merchant session', {
          error: e instanceof Error ? e.message : e,
        });
      }
    }

    if (customerSessionCookie) {
      try {
        const decodedSession = Buffer.from(customerSessionCookie, 'base64').toString('utf-8');
        customerSessionData = JSON.parse(decodedSession);
      } catch (e) {
        logger.debug('Failed to decode customer session', {
          error: e instanceof Error ? e.message : e,
        });
      }
    }

    const debugInfo = {
      timestamp: new Date().toISOString(),
      cookies: {
        hasCypheraSession: !!sessionCookie,
        hasCustomerSession: !!customerSessionCookie,
        sessionPreview: sessionCookie ? sessionCookie.substring(0, 20) + '...' : null,
        customerSessionPreview: customerSessionCookie
          ? customerSessionCookie.substring(0, 20) + '...'
          : null,
      },
      merchantSession: sessionData
        ? {
            valid: true,
            expired: sessionData.expires_at && sessionData.expires_at < Date.now() / 1000,
            email: sessionData.email,
            accountId: sessionData.account_id,
            userId: sessionData.user_id,
            workspaceId: sessionData.workspace_id,
          }
        : null,
      customerSession: customerSessionData
        ? {
            valid: true,
            expired:
              customerSessionData.expires_at && customerSessionData.expires_at < Date.now() / 1000,
            email: customerSessionData.customer_email,
            customerId: customerSessionData.customer_id,
            finishedOnboarding: customerSessionData.finished_onboarding,
          }
        : null,
      environment: {
        nodeEnv: process.env.NODE_ENV,
        runtime: process.env.NEXT_RUNTIME || 'nodejs',
      },
    };

    return NextResponse.json(debugInfo);
  } catch (error) {
    logger.error('Debug session endpoint failed', {
      error: error instanceof Error ? error.message : error,
    });

    return NextResponse.json(
      { error: 'Debug failed', message: error instanceof Error ? error.message : 'Unknown error' },
      { status: 500 }
    );
  }
}
