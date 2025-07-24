import { NextRequest, NextResponse } from 'next/server';
import { getUser } from '@/lib/auth/session/session';
import { createHeadersWithCorrelationId } from '@/lib/utils/correlation';

// GET /api/api-keys/:id - Get API key by ID
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const { id } = await params;
    const user = await getUser();
    if (!user || !user.access_token) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }

    const apiUrl = process.env.CYPHERA_API_BASE_URL;
    const headers = createHeadersWithCorrelationId({
      Authorization: `Bearer ${user.access_token}`,
      'X-Workspace-ID': user.workspace_id || '',
    });

    const response = await fetch(`${apiUrl}/api/v1/api-keys/${id}`, {
      headers,
    });

    const data = await response.json();

    if (!response.ok) {
      return NextResponse.json(data, { status: response.status });
    }

    return NextResponse.json(data);
  } catch (error) {
    console.error('Failed to fetch API key:', error);
    return NextResponse.json(
      { error: 'Failed to fetch API key' },
      { status: 500 }
    );
  }
}

// PUT /api/api-keys/:id - Update API key
export async function PUT(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const { id } = await params;
    const user = await getUser();
    if (!user || !user.access_token) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }

    const body = await request.json();

    const apiUrl = process.env.CYPHERA_API_BASE_URL;
    const headers = createHeadersWithCorrelationId({
      'Content-Type': 'application/json',
      Authorization: `Bearer ${user.access_token}`,
      'X-Workspace-ID': user.workspace_id || '',
    });

    const response = await fetch(`${apiUrl}/api/v1/api-keys/${id}`, {
      method: 'PUT',
      headers,
      body: JSON.stringify(body),
    });

    const data = await response.json();

    if (!response.ok) {
      return NextResponse.json(data, { status: response.status });
    }

    return NextResponse.json(data);
  } catch (error) {
    console.error('Failed to update API key:', error);
    return NextResponse.json(
      { error: 'Failed to update API key' },
      { status: 500 }
    );
  }
}

// DELETE /api/api-keys/:id - Delete API key
export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const { id } = await params;
    const user = await getUser();
    if (!user || !user.access_token) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }

    const apiUrl = process.env.CYPHERA_API_BASE_URL;
    const headers = createHeadersWithCorrelationId({
      Authorization: `Bearer ${user.access_token}`,
      'X-Workspace-ID': user.workspace_id || '',
    });

    const response = await fetch(`${apiUrl}/api/v1/api-keys/${id}`, {
      method: 'DELETE',
      headers,
    });

    if (!response.ok) {
      const data = await response.json();
      return NextResponse.json(data, { status: response.status });
    }

    return new NextResponse(null, { status: 204 });
  } catch (error) {
    console.error('Failed to delete API key:', error);
    return NextResponse.json(
      { error: 'Failed to delete API key' },
      { status: 500 }
    );
  }
}