# Validation Error Handling Guide

This guide explains how to properly handle validation errors returned from the backend API in the frontend application.

## Overview

The backend validation middleware returns errors in a standardized format:

```json
{
  "errors": [
    {
      "field": "email",
      "message": "must be a valid email address"
    },
    {
      "field": "name",
      "message": "must be at least 3 characters"
    }
  ],
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

## Using the Validation Error Hook

The `useValidationErrors` hook provides a complete solution for managing validation errors:

```typescript
import { useValidationErrors } from '@/hooks/use-validation-errors';

function MyComponent() {
  const {
    validationErrors,      // Array of validation errors
    fieldErrors,          // Object mapping field names to error messages
    handleValidationError, // Function to process API error responses
    getFieldError,        // Get error for specific field
    clearValidationErrors, // Clear all errors
    clearFieldError,      // Clear error for specific field
    hasErrors,           // Boolean indicating if there are errors
  } = useValidationErrors();
  
  // ... rest of component
}
```

## Example: Form with Validation

Here's a complete example of a form that handles validation errors:

```typescript
import { useValidationErrors } from '@/hooks/use-validation-errors';
import { ValidationErrorDisplay, InlineValidationError } from '@/components/ui/validation-error';
import { isValidationError } from '@/types/validation';

export function CreateUserForm() {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const {
    validationErrors,
    clearValidationErrors,
    handleValidationError,
    getFieldError,
  } = useValidationErrors();

  const onSubmit = async (data: FormData) => {
    try {
      setIsSubmitting(true);
      clearValidationErrors(); // Clear previous errors
      
      const response = await fetch('/api/users', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });

      if (!response.ok) {
        const errorData = await response.json();
        
        // Check if it's a validation error
        if (response.status === 400 && isValidationError(errorData)) {
          handleValidationError(errorData);
          return; // Stay on form to fix errors
        }
        
        // Handle other errors
        throw new Error(errorData.error || 'Failed to create user');
      }

      // Success handling
      toast.success('User created successfully');
      // Navigate or close dialog
      
    } catch (error) {
      console.error('Error:', error);
      toast.error(error.message);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      {/* Display all validation errors at the top */}
      <ValidationErrorDisplay errors={validationErrors} />
      
      <div>
        <Label htmlFor="email">Email</Label>
        <Input
          id="email"
          type="email"
          {...register('email')}
          // Combine client and server validation errors
          error={errors.email?.message || getFieldError('email')}
        />
        {/* Or show inline error */}
        <InlineValidationError errors={validationErrors} field="email" />
      </div>
      
      {/* More form fields... */}
      
      <Button type="submit" disabled={isSubmitting}>
        Submit
      </Button>
    </form>
  );
}
```

## Components for Displaying Errors

### ValidationErrorDisplay

Shows all validation errors in an alert box:

```typescript
<ValidationErrorDisplay 
  errors={validationErrors} 
  className="mb-4" 
/>
```

### InlineValidationError

Shows error for a specific field inline:

```typescript
<InlineValidationError 
  errors={validationErrors} 
  field="email" 
/>
```

### FieldError

Generic field error display:

```typescript
<FieldError 
  error={getFieldError('email')} 
  className="mt-1" 
/>
```

## Best Practices

1. **Clear errors before submission**: Always call `clearValidationErrors()` before submitting
2. **Check response status**: Validation errors typically come with 400 status
3. **Use type guards**: Use `isValidationError()` to verify error format
4. **Combine validations**: Show both client-side and server-side errors
5. **Clear on close**: Clear errors when dialogs/forms close
6. **Log with correlation ID**: Validation errors include correlation IDs for debugging

## Integration with React Hook Form

When using React Hook Form, combine both validation sources:

```typescript
<Input
  {...register('email')}
  error={
    // Client-side validation from React Hook Form
    formState.errors.email?.message || 
    // Server-side validation from API
    getFieldError('email')
  }
/>
```

## Handling Different Error Types

```typescript
if (!response.ok) {
  const errorData = await response.json();
  
  if (response.status === 400 && isValidationError(errorData)) {
    // Validation errors - show on form
    handleValidationError(errorData);
  } else if (response.status === 429) {
    // Rate limit error
    toast.error('Too many requests. Please try again later.');
  } else {
    // Generic error
    toast.error(errorData.error || 'An error occurred');
  }
}
```

## Testing Validation Errors

When writing E2E tests:

```typescript
test('should show validation errors', async ({ page }) => {
  // Submit invalid data
  await page.fill('input[name="email"]', 'invalid-email');
  await page.click('button[type="submit"]');
  
  // Check for validation error
  await expect(page.locator('text=must be a valid email address')).toBeVisible();
  
  // Fix the error
  await page.fill('input[name="email"]', 'valid@example.com');
  await page.click('button[type="submit"]');
  
  // Should succeed
  await expect(page.locator('text=Success')).toBeVisible();
});
```