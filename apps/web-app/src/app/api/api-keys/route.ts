import { NextRequest, NextResponse } from 'next/server';
import { getUser } from '@/lib/auth/session/session';
import { createHeadersWithCorrelationId } from '@/lib/utils/correlation';
import { withCSRFProtection } from '@/lib/security/csrf-middleware';

// GET /api/api-keys - List API keys
export async function GET(request: NextRequest) {
  try {
    const user = await getUser();
    if (!user || !user.access_token) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }

    const apiUrl = process.env.CYPHERA_API_BASE_URL;
    const headers = createHeadersWithCorrelationId({
      Authorization: `Bearer ${user.access_token}`,
      'X-Workspace-ID': user.workspace_id || '',
    });

    const response = await fetch(`${apiUrl}/api/v1/api-keys`, {
      headers,
    });

    const data = await response.json();

    if (!response.ok) {
      return NextResponse.json(data, { status: response.status });
    }

    return NextResponse.json(data);
  } catch (error) {
    console.error('Failed to fetch API keys:', error);
    return NextResponse.json(
      { error: 'Failed to fetch API keys' },
      { status: 500 }
    );
  }
}

// POST /api/api-keys - Create API key
export const POST = withCSRFProtection(async (request: NextRequest) => {
  try {
    const user = await getUser();
    if (!user || !user.access_token) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }

    const body = await request.json();
    const { name, description, access_level } = body;

    if (!name || !name.trim()) {
      return NextResponse.json(
        { error: 'API key name is required' },
        { status: 400 }
      );
    }

    const apiUrl = process.env.CYPHERA_API_BASE_URL;
    const headers = createHeadersWithCorrelationId({
      'Content-Type': 'application/json',
      Authorization: `Bearer ${user.access_token}`,
      'X-Workspace-ID': user.workspace_id || '',
    });

    const requestBody: any = {
      name: name.trim(),
      access_level: access_level || 'write',
    };
    
    // Only include description if provided
    if (description && description.trim()) {
      requestBody.description = description.trim();
    }
    
    console.log('Sending to backend API:', requestBody);
    
    const response = await fetch(`${apiUrl}/api/v1/api-keys`, {
      method: 'POST',
      headers,
      body: JSON.stringify(requestBody),
    });

    const data = await response.json();

    if (!response.ok) {
      console.error('Backend API error:', {
        status: response.status,
        data,
        requestBody,
      });
      return NextResponse.json(data, { status: response.status });
    }

    return NextResponse.json(data);
  } catch (error) {
    console.error('Failed to create API key:', error);
    return NextResponse.json(
      { error: 'Failed to create API key' },
      { status: 500 }
    );
  }
});