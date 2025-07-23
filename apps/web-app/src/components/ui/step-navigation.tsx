'use client';

import { cn } from '@/lib/utils';
import { Check } from 'lucide-react';

interface Step {
  id: string;
  title: string;
  icon?: React.ReactNode;
  description?: string;
}

interface StepNavigationProps {
  steps: Step[];
  currentStep: number; // 0-indexed
  completedSteps: number[]; // array of completed step indices
  onStepClick?: (stepIndex: number) => void;
  className?: string;
}

export function StepNavigation({
  steps,
  currentStep,
  completedSteps,
  onStepClick,
  className,
}: StepNavigationProps) {
  return (
    <div className={cn('w-full', className)}>
      {/* Progress Bar */}
      <div className="relative mb-8">
        <div className="absolute top-4 left-0 w-full h-0.5 bg-gray-200 dark:bg-gray-700" />
        <div
          className="absolute top-4 left-0 h-0.5 bg-gradient-to-r from-blue-500 to-green-500 transition-all duration-500 ease-in-out"
          style={{
            width: `${(currentStep / (steps.length - 1)) * 100}%`,
          }}
        />

        {/* Step Indicators */}
        <div className="relative flex justify-between">
          {steps.map((step, index) => {
            const isCompleted = completedSteps.includes(index);
            const isCurrent = index === currentStep;
            const isClickable = onStepClick && (isCompleted || index <= currentStep);

            return (
              <button
                key={step.id}
                onClick={() => isClickable && onStepClick(index)}
                disabled={!isClickable}
                className={cn(
                  'flex flex-col items-center group transition-all duration-200',
                  isClickable && 'cursor-pointer hover:scale-105',
                  !isClickable && 'cursor-default'
                )}
              >
                {/* Step Circle */}
                <div
                  className={cn(
                    'w-8 h-8 rounded-full border-2 flex items-center justify-center text-sm font-medium transition-all duration-200',
                    'bg-white dark:bg-gray-900',
                    {
                      // Completed step
                      'border-green-500 text-green-600 bg-green-50 dark:bg-green-900/20':
                        isCompleted,
                      // Current step
                      'border-blue-500 text-blue-600 bg-blue-50 dark:bg-blue-900/20 ring-4 ring-blue-100 dark:ring-blue-900/40':
                        isCurrent && !isCompleted,
                      // Future step
                      'border-gray-300 text-gray-400 dark:border-gray-600 dark:text-gray-500':
                        !isCurrent && !isCompleted,
                    }
                  )}
                >
                  {isCompleted ? <Check className="w-4 h-4" /> : step.icon ? step.icon : index + 1}
                </div>

                {/* Step Label */}
                <div className="mt-2 text-center">
                  <div
                    className={cn('text-xs font-medium transition-colors duration-200', {
                      'text-green-600 dark:text-green-400': isCompleted,
                      'text-blue-600 dark:text-blue-400': isCurrent && !isCompleted,
                      'text-gray-500 dark:text-gray-400': !isCurrent && !isCompleted,
                    })}
                  >
                    {step.title}
                  </div>
                  {step.description && (
                    <div className="text-xs text-gray-400 mt-1 max-w-20">{step.description}</div>
                  )}
                </div>
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}
