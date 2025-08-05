import { Badge } from '@/components/ui/badge';
import type { InvoiceStatus } from '@/types/invoice';

interface InvoiceStatusBadgeProps {
  status: InvoiceStatus;
}

export function InvoiceStatusBadge({ status }: InvoiceStatusBadgeProps) {
  const getStatusConfig = (status: InvoiceStatus) => {
    switch (status) {
      case 'draft':
        return {
          label: 'Draft',
          className: 'bg-gray-100 text-gray-800 hover:bg-gray-200',
        };
      case 'open':
        return {
          label: 'Open',
          className: 'bg-blue-100 text-blue-800 hover:bg-blue-200',
        };
      case 'paid':
        return {
          label: 'Paid',
          className: 'bg-green-100 text-green-800 hover:bg-green-200',
        };
      case 'void':
        return {
          label: 'Void',
          className: 'bg-red-100 text-red-800 hover:bg-red-200',
        };
      case 'uncollectible':
        return {
          label: 'Uncollectible',
          className: 'bg-orange-100 text-orange-800 hover:bg-orange-200',
        };
      default:
        return {
          label: status,
          className: 'bg-gray-100 text-gray-800 hover:bg-gray-200',
        };
    }
  };

  const config = getStatusConfig(status);

  return (
    <Badge className={config.className} variant="secondary">
      {config.label}
    </Badge>
  );
}