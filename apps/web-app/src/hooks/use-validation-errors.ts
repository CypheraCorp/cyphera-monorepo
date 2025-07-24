import { useState, useCallback } from 'react';
import { 
  ValidationError, 
  ValidationErrorResponse, 
  isValidationError,
  validationErrorsToMap,
  getFieldError 
} from '@/types/validation';
import { useCorrelationId } from '@/hooks/utils/use-correlation-id';

interface UseValidationErrorsReturn {
  validationErrors: ValidationError[];
  fieldErrors: Record<string, string>;
  setValidationErrors: (errors: ValidationError[]) => void;
  clearValidationErrors: () => void;
  clearFieldError: (field: string) => void;
  handleValidationError: (error: any) => boolean;
  getFieldError: (field: string) => string | undefined;
  hasErrors: boolean;
}

/**
 * Hook for managing validation errors from the backend
 */
export function useValidationErrors(): UseValidationErrorsReturn {
  const [validationErrors, setValidationErrors] = useState<ValidationError[]>([]);
  const { logError } = useCorrelationId();

  // Convert to field map for easy lookup
  const fieldErrors = validationErrorsToMap(validationErrors);

  // Clear all validation errors
  const clearValidationErrors = useCallback(() => {
    setValidationErrors([]);
  }, []);

  // Clear error for a specific field
  const clearFieldError = useCallback((field: string) => {
    setValidationErrors(prev => prev.filter(e => e.field !== field));
  }, []);

  // Handle validation error from API response
  const handleValidationError = useCallback((error: any): boolean => {
    if (isValidationError(error)) {
      setValidationErrors(error.errors);
      
      // Log with correlation ID for debugging
      logError('Validation errors received', error, {
        errorCount: error.errors.length,
        fields: error.errors.map(e => e.field),
      });
      
      return true;
    }
    return false;
  }, [logError]);

  // Get error for a specific field
  const getFieldErrorForField = useCallback((field: string): string | undefined => {
    return getFieldError(validationErrors, field);
  }, [validationErrors]);

  return {
    validationErrors,
    fieldErrors,
    setValidationErrors,
    clearValidationErrors,
    clearFieldError,
    handleValidationError,
    getFieldError: getFieldErrorForField,
    hasErrors: validationErrors.length > 0,
  };
}