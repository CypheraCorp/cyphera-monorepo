// Shared utility functions

export function formatDate(date: Date): string {
  return date.toISOString().split('T')[0];
}

export function delay(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}