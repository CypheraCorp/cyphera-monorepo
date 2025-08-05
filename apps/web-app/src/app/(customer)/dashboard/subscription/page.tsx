'use client';

import React from 'react';
import { useSearchParams } from 'next/navigation';
import { SubscriptionManagement } from '@/components/subscriptions/subscription-management';
import { Card, CardContent } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { ArrowLeft, AlertCircle } from 'lucide-react';
import Link from 'next/link';

export default function SubscriptionManagementPage() {
  const searchParams = useSearchParams();
  const subscriptionId = searchParams.get('id');

  if (!subscriptionId) {
    return (
      <div className="container max-w-4xl mx-auto py-8 px-4">
        <Card>
          <CardContent className="py-12">
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>
                No subscription ID provided. Please select a subscription to manage.
              </AlertDescription>
            </Alert>
            <div className="mt-6 text-center">
              <Link href="/dashboard">
                <Button variant="outline">
                  <ArrowLeft className="mr-2 h-4 w-4" />
                  Back to Dashboard
                </Button>
              </Link>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="container max-w-4xl mx-auto py-8 px-4">
      <div className="mb-6">
        <Link href="/dashboard">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Dashboard
          </Button>
        </Link>
      </div>

      <SubscriptionManagement subscriptionId={subscriptionId} />
    </div>
  );
}