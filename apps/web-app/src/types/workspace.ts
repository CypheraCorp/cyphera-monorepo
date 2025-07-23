/**
 * Response structure for workspace data
 */
export interface WorkspaceResponse {
  id: string;
  object: string;
  name: string;
  description?: string;
  business_name: string;
  business_type?: string;
  website_url?: string;
  support_email?: string;
  support_phone?: string;
  account_id: string;
  metadata?: Record<string, unknown>;
  livemode: boolean;
  created: number;
  updated: number;
}
