import { AlertCircle } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { ValidationError } from '@/types/validation';
import { cn } from '@/lib/utils';

interface ValidationErrorDisplayProps {
  errors: ValidationError[];
  className?: string;
}

/**
 * Display validation errors in an alert
 */
export function ValidationErrorDisplay({ errors, className }: ValidationErrorDisplayProps) {
  if (errors.length === 0) return null;

  return (
    <Alert variant="destructive" className={cn('mb-4', className)}>
      <AlertCircle className="h-4 w-4" />
      <AlertDescription>
        {errors.length === 1 ? (
          <span>{errors[0].message}</span>
        ) : (
          <ul className="list-disc list-inside space-y-1">
            {errors.map((error, index) => (
              <li key={`${error.field}-${index}`}>
                <strong>{error.field}:</strong> {error.message}
              </li>
            ))}
          </ul>
        )}
      </AlertDescription>
    </Alert>
  );
}

interface FieldErrorProps {
  error?: string;
  className?: string;
}

/**
 * Display error for a specific field
 */
export function FieldError({ error, className }: FieldErrorProps) {
  if (!error) return null;

  return (
    <p className={cn('text-sm text-destructive mt-1', className)}>
      {error}
    </p>
  );
}

interface InlineValidationErrorProps {
  errors: ValidationError[];
  field: string;
  className?: string;
}

/**
 * Display inline validation error for a specific field
 */
export function InlineValidationError({ errors, field, className }: InlineValidationErrorProps) {
  const error = errors.find(e => e.field === field);
  
  if (!error) return null;

  return <FieldError error={error.message} className={className} />;
}