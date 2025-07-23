'use client';

import { Building2 } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { CompanyRegistrationForm } from '@/components/auth/company-registration-form';
import type { CypheraUser } from '@/lib/auth/session/session';
import { hasRequiredAccountInfo } from '@/lib/auth/guards/user-guards';

interface SettingsFormProps {
  user: CypheraUser;
}

/**
 * SettingsForm component
 * Displays a tabbed interface for managing various account settings
 */
export function SettingsForm({ user }: SettingsFormProps) {
  // Early return if we don't have required user data
  if (!hasRequiredAccountInfo(user)) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Account Setup Required</CardTitle>
          <CardDescription>
            Your account setup is incomplete. Please try signing out and back in.
          </CardDescription>
        </CardHeader>
      </Card>
    );
  }

  // Create a properly typed user object for the form
  const typedUser = {
    ...user,
    user_id: user.user_id!,
    account_id: user.account_id!,
    workspace_id: user.workspace_id!,
  };

  return (
    <Tabs defaultValue="account" className="space-y-6">
      <TabsList>
        <TabsTrigger value="account" className="flex items-center gap-2">
          <Building2 className="h-4 w-4" />
          Account
        </TabsTrigger>
      </TabsList>

      <TabsContent value="account" className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>Account Information</CardTitle>
            <CardDescription>
              Complete your account setup and manage your company information
            </CardDescription>
          </CardHeader>
          <CardContent>
            <CompanyRegistrationForm user={typedUser} />
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  );
}
