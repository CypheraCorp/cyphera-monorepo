/**
 * Validation error types matching backend validation middleware
 */

export interface ValidationError {
  field: string;
  message: string;
}

export interface ValidationErrorResponse {
  errors: ValidationError[];
  correlation_id?: string;
}

/**
 * Type guard to check if an error is a validation error response
 */
export function isValidationError(error: any): error is ValidationErrorResponse {
  return (
    error &&
    typeof error === 'object' &&
    'errors' in error &&
    Array.isArray(error.errors) &&
    error.errors.every((e: any) => 
      typeof e === 'object' && 
      'field' in e && 
      'message' in e
    )
  );
}

/**
 * Convert validation errors to a map for easy field lookup
 */
export function validationErrorsToMap(errors: ValidationError[]): Record<string, string> {
  return errors.reduce((acc, error) => {
    acc[error.field] = error.message;
    return acc;
  }, {} as Record<string, string>);
}

/**
 * Get error message for a specific field
 */
export function getFieldError(errors: ValidationError[], field: string): string | undefined {
  const error = errors.find(e => e.field === field);
  return error?.message;
}

/**
 * Format validation errors for display
 */
export function formatValidationErrors(errors: ValidationError[]): string {
  if (errors.length === 0) return '';
  if (errors.length === 1) return errors[0].message;
  
  return errors
    .map(e => `${e.field}: ${e.message}`)
    .join(', ');
}