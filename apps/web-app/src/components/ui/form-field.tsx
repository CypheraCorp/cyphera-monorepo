'use client';

import { forwardRef } from 'react';
import { Controller, useFormContext } from 'react-hook-form';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { cn } from '@/lib/utils';
import { AlertCircle } from 'lucide-react';

// Base field wrapper component
interface FieldWrapperProps {
  label?: string;
  error?: string;
  required?: boolean;
  className?: string;
  children: React.ReactNode;
}

function FieldWrapper({ label, error, required, className, children }: FieldWrapperProps) {
  return (
    <div className={cn('space-y-2', className)}>
      {label && (
        <Label className={cn(error && 'text-destructive')}>
          {label}
          {required && <span className="text-destructive ml-1">*</span>}
        </Label>
      )}
      {children}
      {error && (
        <div className="flex items-center gap-1 text-sm text-destructive">
          <AlertCircle className="h-3 w-3" />
          <span>{error}</span>
        </div>
      )}
    </div>
  );
}

// Text field component
interface TextFieldProps extends React.InputHTMLAttributes<HTMLInputElement> {
  name: string;
  label?: string;
  required?: boolean;
}

export const TextField = forwardRef<HTMLInputElement, TextFieldProps>(
  ({ name, label, required, className, ...props }, ref) => {
    const {
      formState: { errors },
    } = useFormContext();
    const error = errors[name]?.message as string | undefined;

    return (
      <Controller
        name={name}
        render={({ field }) => (
          <FieldWrapper label={label} error={error} required={required}>
            <Input
              {...field}
              {...props}
              ref={ref}
              className={cn(error && 'border-destructive', className)}
              aria-invalid={!!error}
              aria-describedby={error ? `${name}-error` : undefined}
            />
          </FieldWrapper>
        )}
      />
    );
  }
);
TextField.displayName = 'TextField';

// Textarea field component
interface TextAreaFieldProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  name: string;
  label?: string;
  required?: boolean;
}

export const TextAreaField = forwardRef<HTMLTextAreaElement, TextAreaFieldProps>(
  ({ name, label, required, className, ...props }, ref) => {
    const {
      formState: { errors },
    } = useFormContext();
    const error = errors[name]?.message as string | undefined;

    return (
      <Controller
        name={name}
        render={({ field }) => (
          <FieldWrapper label={label} error={error} required={required}>
            <Textarea
              {...field}
              {...props}
              ref={ref}
              className={cn(error && 'border-destructive', className)}
              aria-invalid={!!error}
              aria-describedby={error ? `${name}-error` : undefined}
            />
          </FieldWrapper>
        )}
      />
    );
  }
);
TextAreaField.displayName = 'TextAreaField';

// Select field component
interface SelectFieldProps {
  name: string;
  label?: string;
  required?: boolean;
  placeholder?: string;
  options: Array<{ label: string; value: string }>;
  className?: string;
}

export function SelectField({
  name,
  label,
  required,
  placeholder,
  options,
  className,
}: SelectFieldProps) {
  const {
    formState: { errors },
  } = useFormContext();
  const error = errors[name]?.message as string | undefined;

  return (
    <Controller
      name={name}
      render={({ field }) => (
        <FieldWrapper label={label} error={error} required={required} className={className}>
          <Select onValueChange={field.onChange} defaultValue={field.value}>
            <SelectTrigger className={cn(error && 'border-destructive')}>
              <SelectValue placeholder={placeholder} />
            </SelectTrigger>
            <SelectContent>
              {options.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </FieldWrapper>
      )}
    />
  );
}

// Checkbox field component
interface CheckboxFieldProps {
  name: string;
  label?: string;
  description?: string;
  className?: string;
}

export function CheckboxField({ name, label, description, className }: CheckboxFieldProps) {
  const {
    formState: { errors },
  } = useFormContext();
  const error = errors[name]?.message as string | undefined;

  return (
    <Controller
      name={name}
      render={({ field }) => (
        <div className={cn('space-y-2', className)}>
          <div className="flex items-start space-x-3">
            <Checkbox
              checked={field.value}
              onCheckedChange={field.onChange}
              id={name}
              className={cn(error && 'border-destructive')}
            />
            <div className="space-y-1 leading-none">
              {label && (
                <Label htmlFor={name} className="font-normal cursor-pointer">
                  {label}
                </Label>
              )}
              {description && <p className="text-sm text-muted-foreground">{description}</p>}
            </div>
          </div>
          {error && (
            <div className="flex items-center gap-1 text-sm text-destructive">
              <AlertCircle className="h-3 w-3" />
              <span>{error}</span>
            </div>
          )}
        </div>
      )}
    />
  );
}

// Radio group field component
interface RadioGroupFieldProps {
  name: string;
  label?: string;
  required?: boolean;
  options: Array<{ label: string; value: string; description?: string }>;
  className?: string;
}

export function RadioGroupField({
  name,
  label,
  required,
  options,
  className,
}: RadioGroupFieldProps) {
  const {
    formState: { errors },
  } = useFormContext();
  const error = errors[name]?.message as string | undefined;

  return (
    <Controller
      name={name}
      render={({ field }) => (
        <FieldWrapper label={label} error={error} required={required} className={className}>
          <RadioGroup
            onValueChange={field.onChange}
            defaultValue={field.value}
            className="space-y-2"
          >
            {options.map((option) => (
              <div key={option.value} className="flex items-start space-x-3">
                <RadioGroupItem value={option.value} id={`${name}-${option.value}`} />
                <div className="space-y-1 leading-none">
                  <Label htmlFor={`${name}-${option.value}`} className="font-normal cursor-pointer">
                    {option.label}
                  </Label>
                  {option.description && (
                    <p className="text-sm text-muted-foreground">{option.description}</p>
                  )}
                </div>
              </div>
            ))}
          </RadioGroup>
        </FieldWrapper>
      )}
    />
  );
}
