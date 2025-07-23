/**
 * Debug API Test Endpoint
 * 
 * This endpoint helps debug API connectivity and environment variable access
 */

import { NextRequest, NextResponse } from 'next/server';

export async function GET(request: NextRequest) {
  try {
    // Test environment variables
    const envCheck = {
      CYPHERA_API_KEY: process.env.CYPHERA_API_KEY ? '✅ Set' : '❌ Missing',
      CYPHERA_API_BASE_URL: process.env.CYPHERA_API_BASE_URL || 'Not set',
      NODE_ENV: process.env.NODE_ENV || 'Not set',
      NEXT_PUBLIC_ENABLE_DEV_AUTH_BYPASS: process.env.NEXT_PUBLIC_ENABLE_DEV_AUTH_BYPASS || 'Not set',
    };

    // Test network fetch
    let networkTest = 'Not tested';
    try {
      const baseUrl = process.env.CYPHERA_API_BASE_URL || 'http://localhost:8000';
      const apiUrl = baseUrl.endsWith('/api/v1') ? baseUrl : `${baseUrl}/api/v1`;
      const networkUrl = `${apiUrl}/networks?active=true`;
      
      const response = await fetch(networkUrl, {
        method: 'GET',
        headers: {
          'X-API-Key': process.env.CYPHERA_API_KEY || '',
          'Content-Type': 'application/json',
          Accept: 'application/json',
        },
      });

      if (response.ok) {
        const data = await response.json();
        networkTest = `✅ Success - ${data.data?.length || 0} networks`;
      } else {
        networkTest = `❌ Failed - ${response.status} ${response.statusText}`;
      }
    } catch (_error) {
      networkTest = `❌ Error - ${_error instanceof Error ? _error.message : 'Unknown error'}`;
    }

    return NextResponse.json({
      timestamp: new Date().toISOString(),
      environment: envCheck,
      networkApiTest: networkTest,
      url: request.url,
      headers: Object.fromEntries(request.headers.entries()),
    });

  } catch (error) {
    console.error('API test error:', error);
    return NextResponse.json(
      { 
        error: 'API test failed', 
        message: error instanceof Error ? error.message : 'Unknown error' 
      },
      { status: 500 }
    );
  }
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    
    return NextResponse.json({
      message: 'API test endpoint is working',
      receivedData: body,
      timestamp: new Date().toISOString(),
    });
  } catch (_error) {
    return NextResponse.json(
      { error: 'Failed to process request' },
      { status: 400 }
    );
  }
}