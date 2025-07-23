import { NextRequest, NextResponse } from 'next/server';
import { logger } from '@/lib/core/logger/logger';

/**
 * PUT /api/customers/[customerId]
 * Update customer details including onboarding status
 */
export async function PUT(
  request: NextRequest,
  { params }: { params: Promise<{ customerId: string }> }
) {
  try {
    const { customerId } = await params;

    // Get session from cookie
    const sessionCookie = request.cookies.get('cyphera-customer-session')?.value;

    if (!sessionCookie) {
      return NextResponse.json({ error: 'No customer session found' }, { status: 401 });
    }

    // Define customer session data type
    interface CustomerSessionData {
      customer_id: string;
      customer_email: string;
      customer_name?: string;
      finished_onboarding?: boolean;
      expires_at?: number;
      [key: string]: unknown;
    }

    // Decode session data from cookie
    let sessionData: CustomerSessionData;
    try {
      const decodedSession = Buffer.from(sessionCookie, 'base64').toString('utf-8');
      sessionData = JSON.parse(decodedSession) as CustomerSessionData;

      // Check if session is expired
      if (sessionData.expires_at && sessionData.expires_at < Date.now() / 1000) {
        return NextResponse.json({ error: 'Session expired' }, { status: 401 });
      }
    } catch (_error) {
      return NextResponse.json({ error: 'Invalid session format' }, { status: 401 });
    }

    // Verify the customer ID matches the session
    if (sessionData.customer_id !== customerId) {
      return NextResponse.json({ error: 'Unauthorized to update this customer' }, { status: 403 });
    }

    // Parse request body
    const updateData = await request.json();

    logger.info('Updating customer', {
      customerId,
      updateData,
      sessionEmail: sessionData.customer_email,
    });

    try {
      // For now, we'll skip the backend API call and just update the session
      // In a real implementation, you would call the backend API here
      logger.debug('Updating customer (mock implementation)', {
        customerId,
        updateData,
        sessionEmail: sessionData.customer_email,
      });

      // Create mock response
      const mockResponse = {
        id: customerId,
        object: 'customer',
        email: sessionData.customer_email,
        name: updateData.name || sessionData.customer_name,
        phone: updateData.phone,
        description: updateData.description,
        finished_onboarding:
          updateData.finished_onboarding ?? sessionData.finished_onboarding ?? false,
        updated_at: Math.floor(Date.now() / 1000),
      };

      // Update session data with new values
      const updatedSessionData = { ...sessionData };
      if (updateData.finished_onboarding !== undefined) {
        updatedSessionData.finished_onboarding = updateData.finished_onboarding;
      }
      if (updateData.name) {
        updatedSessionData.customer_name = updateData.name;
      }

      // Create response and update the session cookie with new data
      const response = NextResponse.json(mockResponse);

      // Update session cookie with new data
      const encodedSession = Buffer.from(JSON.stringify(updatedSessionData)).toString('base64');
      response.cookies.set('cyphera-customer-session', encodedSession, {
        httpOnly: true,
        secure: process.env.NODE_ENV === 'production',
        sameSite: 'lax',
        maxAge: 60 * 60 * 24 * 7, // 7 days
        path: '/',
      });

      logger.debug('Updated customer session cookie with new data');
      logger.debug('Mock customer update response', { mockResponse });
      return response;
    } catch (backendError) {
      logger.error('Customer update failed', {
        error: backendError instanceof Error ? backendError.message : backendError,
      });
      return NextResponse.json({ error: 'Failed to update customer' }, { status: 500 });
    }
  } catch (error) {
    logger.error('Customer update API error', {
      error: error instanceof Error ? error.message : error,
    });
    const message = error instanceof Error ? error.message : 'Failed to update customer';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
