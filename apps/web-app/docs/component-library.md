# Cyphera Component Library Documentation

## Overview

This document describes the standardized component library for the Cyphera Web application. All components follow consistent patterns and are built with accessibility and performance in mind.

## Table of Contents

1. [Form Components](#form-components)
2. [Data Table](#data-table)
3. [Loading States](#loading-states)
4. [Error Boundaries](#error-boundaries)
5. [Loading Management](#loading-management)
6. [Optimistic Updates](#optimistic-updates)

## Form Components

### TextField

A standardized text input field with built-in validation and error handling.

```tsx
import { TextField } from '@/components/ui/form-field';

<TextField
  name="email"
  label="Email Address"
  type="email"
  placeholder="user@example.com"
  required
/>;
```

### TextAreaField

Multi-line text input with validation support.

```tsx
import { TextAreaField } from '@/components/ui/form-field';

<TextAreaField
  name="description"
  label="Description"
  placeholder="Enter description..."
  rows={4}
/>;
```

### SelectField

Dropdown selection field with options.

```tsx
import { SelectField } from '@/components/ui/form-field';

<SelectField
  name="role"
  label="User Role"
  required
  options={[
    { label: 'Admin', value: 'admin' },
    { label: 'User', value: 'user' },
  ]}
/>;
```

### CheckboxField

Checkbox input with optional description.

```tsx
import { CheckboxField } from '@/components/ui/form-field';

<CheckboxField
  name="terms"
  label="I agree to the terms and conditions"
  description="You must agree to continue"
/>;
```

### RadioGroupField

Radio button group for single selection.

```tsx
import { RadioGroupField } from '@/components/ui/form-field';

<RadioGroupField
  name="plan"
  label="Select Plan"
  options={[
    { label: 'Free', value: 'free', description: 'Basic features' },
    { label: 'Pro', value: 'pro', description: '$10/month' },
  ]}
/>;
```

### Usage with React Hook Form

All form components are designed to work seamlessly with React Hook Form:

```tsx
import { useForm, FormProvider } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';

const schema = z.object({
  name: z.string().min(2),
  email: z.string().email(),
});

function MyForm() {
  const methods = useForm({
    resolver: zodResolver(schema),
  });

  return (
    <FormProvider {...methods}>
      <form onSubmit={methods.handleSubmit(onSubmit)}>
        <TextField name="name" label="Name" required />
        <TextField name="email" label="Email" type="email" required />
        <Button type="submit">Submit</Button>
      </form>
    </FormProvider>
  );
}
```

## Data Table

A powerful, reusable table component with built-in features.

### Features

- Column sorting
- Global and column-specific filtering
- Pagination
- Column visibility toggle
- Loading states
- Row click handlers

### Basic Usage

```tsx
import { DataTable } from '@/components/ui/data-table';
import { ColumnDef } from '@tanstack/react-table';

const columns: ColumnDef<User>[] = [
  {
    accessorKey: 'name',
    header: 'Name',
  },
  {
    accessorKey: 'email',
    header: 'Email',
  },
];

<DataTable columns={columns} data={users} searchPlaceholder="Search users..." searchKey="name" />;
```

### Advanced Usage

```tsx
<DataTable
  columns={columns}
  data={users}
  searchPlaceholder="Search..."
  showColumnVisibility={true}
  showPagination={true}
  isLoading={isLoading}
  pageSize={20}
  onRowClick={(row) => console.log('Clicked:', row)}
/>
```

## Loading States

Pre-built loading components for consistent UX.

### LoadingSpinner

Centered spinner with optional message.

```tsx
import { LoadingSpinner } from '@/components/ui/loading-states';

<LoadingSpinner message="Loading data..." />;
```

### Skeleton Loaders

Various skeleton components for different UI patterns:

```tsx
import {
  TableSkeleton,
  CardSkeleton,
  ProductCardSkeleton,
  ListSkeleton,
  FormSkeleton,
  BalanceCardSkeleton,
  StatsCardSkeleton,
  PageLoadingSkeleton
} from '@/components/ui/loading-states';

// Table with 5 rows and 4 columns
<TableSkeleton rows={5} columns={4} />

// Card placeholder
<CardSkeleton />

// List with 3 items
<ListSkeleton items={3} />

// Form with 4 fields
<FormSkeleton fields={4} />

// Full page loading
<PageLoadingSkeleton title="Loading Dashboard..." />
```

## Error Boundaries

Graceful error handling with fallback UI.

### ErrorBoundary

Basic error boundary with customizable fallback.

```tsx
import { ErrorBoundary } from '@/components/ui/error-boundary';

<ErrorBoundary
  fallback={<div>Something went wrong!</div>}
  onError={(error, errorInfo) => {
    console.error('Error caught:', error);
  }}
>
  <YourComponent />
</ErrorBoundary>;
```

### PageErrorBoundary

Full-page error boundary with styled fallback.

```tsx
import { PageErrorBoundary } from '@/components/ui/error-boundary';

export default function Page() {
  return (
    <PageErrorBoundary>
      <YourPageContent />
    </PageErrorBoundary>
  );
}
```

### AsyncErrorBoundary

Combines error boundary with Suspense.

```tsx
import { AsyncErrorBoundary } from '@/components/ui/error-boundary';

<AsyncErrorBoundary fallback={<CustomError />}>
  <YourAsyncComponent />
</AsyncErrorBoundary>;
```

## Loading Management

### Global Progress Bar

NProgress integration for navigation and API calls.

```tsx
// Already integrated in the root layout
// Manual control:
import { startProgress, stopProgress, setProgress } from '@/components/ui/nprogress';

startProgress();
// ... async operation
stopProgress();
```

### Loading Manager

Centralized loading state management.

```tsx
import { fetchWithProgress } from '@/lib/utils/loading-manager';

// Fetch with automatic progress bar
const data = await fetchWithProgress<User[]>('/api/users');
```

### useLoadingState Hook

Component-level loading state management.

```tsx
import { useLoadingState } from '@/lib/utils/loading-manager';

function MyComponent() {
  const { isLoading, error, execute } = useLoadingState();

  const handleSubmit = async () => {
    await execute(fetch('/api/submit', { method: 'POST' }));
  };

  return (
    <Button onClick={handleSubmit} disabled={isLoading}>
      {isLoading ? 'Submitting...' : 'Submit'}
    </Button>
  );
}
```

## Optimistic Updates

Utilities for implementing optimistic UI updates.

### useOptimisticUpdate Hook

```tsx
import { useOptimisticUpdate } from '@/lib/utils/optimistic-updates';

function TodoList() {
  const { update } = useOptimisticUpdate<Todo[]>();

  const handleDelete = async (id: string) => {
    await update(
      {
        queryKey: ['todos'],
        updateFn: (oldData, { id }) => oldData?.filter((todo) => todo.id !== id) || [],
      },
      { id },
      async () => {
        await fetch(`/api/todos/${id}`, { method: 'DELETE' });
        return updatedTodos;
      }
    );
  };
}
```

### Utility Functions

```tsx
import {
  optimisticDeleteFromList,
  optimisticAddToList,
  optimisticUpdateInList,
  optimisticToggleInList,
} from '@/lib/utils/optimistic-updates';

// Delete item
const newList = optimisticDeleteFromList(oldList, itemId);

// Add item
const newList = optimisticAddToList(oldList, newItem, 'start');

// Update item
const newList = optimisticUpdateInList(oldList, itemId, { name: 'New Name' });

// Toggle boolean field
const newList = optimisticToggleInList(oldList, itemId, 'completed');
```

## Best Practices

1. **Always use FormProvider** with form components for proper validation
2. **Wrap pages in PageErrorBoundary** for graceful error handling
3. **Use skeleton loaders** during data fetching for better UX
4. **Implement optimistic updates** for instant feedback
5. **Use the DataTable component** for all tabular data displays
6. **Leverage useLoadingState** for consistent loading states

## Examples

See `/src/components/examples/component-showcase.tsx` for a complete working example of all components.
