import { NextRequest, NextResponse } from 'next/server';
import { ZodError, ZodSchema } from 'zod';
import { logger } from '@/lib/core/logger/logger-utils';

/**
 * Validation error response
 */
export interface ValidationErrorResponse {
  error: string;
  details: {
    field: string;
    message: string;
  }[];
}

/**
 * Format Zod validation errors into a user-friendly format
 */
export function formatZodError(error: ZodError): ValidationErrorResponse {
  const details = error.errors.map((err) => ({
    field: err.path.join('.'),
    message: err.message,
  }));

  return {
    error: 'Validation failed',
    details,
  };
}

/**
 * Validate request body against a Zod schema
 */
export async function validateBody<T>(
  request: NextRequest,
  schema: ZodSchema<T>
): Promise<{ data: T | null; error: NextResponse | null }> {
  try {
    const body = await request.json();
    const data = schema.parse(body);
    return { data, error: null };
  } catch (error) {
    if (error instanceof ZodError) {
      logger.warn('Request validation failed', {
        path: request.nextUrl.pathname,
        errors: error.errors,
      });
      return {
        data: null,
        error: NextResponse.json(formatZodError(error), { status: 400 }),
      };
    }

    // Handle JSON parse errors
    if (error instanceof SyntaxError) {
      logger.warn('Invalid JSON in request body', {
        path: request.nextUrl.pathname,
        error: error.message,
      });
      return {
        data: null,
        error: NextResponse.json(
          { error: 'Invalid JSON in request body' },
          { status: 400 }
        ),
      };
    }

    // Unknown error
    logger.error('Unknown error during validation', error);
    return {
      data: null,
      error: NextResponse.json(
        { error: 'Internal server error' },
        { status: 500 }
      ),
    };
  }
}

/**
 * Validate query parameters against a Zod schema
 */
export function validateQuery<T>(
  request: NextRequest,
  schema: ZodSchema<T>
): { data: T | null; error: NextResponse | null } {
  try {
    const { searchParams } = new URL(request.url);
    const params = Object.fromEntries(searchParams.entries());
    const data = schema.parse(params);
    return { data, error: null };
  } catch (error) {
    if (error instanceof ZodError) {
      logger.warn('Query parameter validation failed', {
        path: request.nextUrl.pathname,
        errors: error.errors,
      });
      return {
        data: null,
        error: NextResponse.json(formatZodError(error), { status: 400 }),
      };
    }

    // Unknown error
    logger.error('Unknown error during query validation', error);
    return {
      data: null,
      error: NextResponse.json(
        { error: 'Internal server error' },
        { status: 500 }
      ),
    };
  }
}

/**
 * Validate route parameters against a Zod schema
 */
export function validateParams<T>(
  params: unknown,
  schema: ZodSchema<T>
): { data: T | null; error: NextResponse | null } {
  try {
    const data = schema.parse(params);
    return { data, error: null };
  } catch (error) {
    if (error instanceof ZodError) {
      logger.warn('Route parameter validation failed', {
        errors: error.errors,
      });
      return {
        data: null,
        error: NextResponse.json(formatZodError(error), { status: 400 }),
      };
    }

    // Unknown error
    logger.error('Unknown error during param validation', error);
    return {
      data: null,
      error: NextResponse.json(
        { error: 'Internal server error' },
        { status: 500 }
      ),
    };
  }
}

/**
 * Higher-order function to wrap API route handlers with validation
 */
// RouteContext type to match Next.js expectations
type RouteContext = {
  params: Promise<Record<string, string>>;
};

export function withValidation<TBody = unknown, TQuery = unknown, TParams = unknown>(
  config: {
    bodySchema?: ZodSchema<TBody>;
    querySchema?: ZodSchema<TQuery>;
    paramsSchema?: ZodSchema<TParams>;
  },
  handler: (
    request: NextRequest,
    context: {
      body?: TBody;
      query?: TQuery;
      params?: TParams;
    }
  ) => Promise<NextResponse>
) {
  // For routes without dynamic segments
  if (!config.paramsSchema) {
    return async (request: NextRequest): Promise<NextResponse> => {
      const context: {
        body?: TBody;
        query?: TQuery;
        params?: TParams;
      } = {};

      // Validate body if schema is provided
      if (config.bodySchema) {
        const { data, error } = await validateBody(request, config.bodySchema);
        if (error) return error;
        context.body = data!;
      }

      // Validate query if schema is provided
      if (config.querySchema) {
        const { data, error } = validateQuery(request, config.querySchema);
        if (error) return error;
        context.query = data!;
      }

      // Call the handler with validated data
      return handler(request, context);
    };
  }

  // For routes with dynamic segments
  return async (
    request: NextRequest,
    routeContext: RouteContext
  ): Promise<NextResponse> => {
    const context: {
      body?: TBody;
      query?: TQuery;
      params?: TParams;
    } = {};

    // Validate body if schema is provided
    if (config.bodySchema) {
      const { data, error } = await validateBody(request, config.bodySchema);
      if (error) return error;
      context.body = data!;
    }

    // Validate query if schema is provided
    if (config.querySchema) {
      const { data, error } = validateQuery(request, config.querySchema);
      if (error) return error;
      context.query = data!;
    }

    // Validate params if schema is provided
    if (config.paramsSchema && routeContext?.params) {
      const params = await routeContext.params;
      const { data, error } = validateParams(params, config.paramsSchema);
      if (error) return error;
      context.params = data!;
    }

    // Call the handler with validated data
    return handler(request, context);
  };
}