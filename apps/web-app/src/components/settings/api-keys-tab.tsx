'use client';

import { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { useToast } from '@/components/ui/use-toast';
import { Plus, Copy, Trash2, AlertTriangle, Key, Check } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';
import { useCorrelationId } from '@/hooks/utils/use-correlation-id';
import { useCSRF } from '@/hooks/security/use-csrf';

interface ApiKey {
  id: string;
  name: string;
  key_prefix: string;
  access_level: 'read' | 'write' | 'admin';
  created_at: string;
  last_used_at?: string;
  expires_at?: string;
}

interface ApiKeysTabProps {
  workspaceId: string;
  accessToken: string;
}

export function ApiKeysTab({ workspaceId, accessToken }: ApiKeysTabProps) {
  const [apiKeys, setApiKeys] = useState<ApiKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [selectedKey, setSelectedKey] = useState<ApiKey | null>(null);
  const [newKeyData, setNewKeyData] = useState<{
    name: string;
    description: string;
    access_level: 'read' | 'write' | 'admin';
  }>({
    name: '',
    description: '',
    access_level: 'write',
  });
  const [createdKey, setCreatedKey] = useState<{ id: string; key: string } | null>(null);
  const [copiedKeyId, setCopiedKeyId] = useState<string | null>(null);
  
  const { logError } = useCorrelationId();
  const { toast } = useToast();
  const { addCSRFHeader } = useCSRF();

  // Load API keys
  useEffect(() => {
    loadApiKeys();
  }, []);

  const loadApiKeys = async () => {
    try {
      const response = await fetch('/api/api-keys', {
        headers: {
          Authorization: `Bearer ${accessToken}`,
          'X-Workspace-ID': workspaceId,
        },
      });

      if (!response.ok) {
        const error = await response.json();
        throw error;
      }

      const data = await response.json();
      setApiKeys(data.data || []);
    } catch (error) {
      logError('Failed to load API keys', error);
      toast({
        title: 'Error',
        description: 'Failed to load API keys. Please try again.',
        variant: 'destructive',
      });
    } finally {
      setLoading(false);
    }
  };

  const handleCreateKey = async () => {
    if (!newKeyData.name.trim()) {
      toast({
        title: 'Error',
        description: 'Please enter a name for the API key.',
        variant: 'destructive',
      });
      return;
    }

    setCreating(true);
    try {
      const payload = {
        name: newKeyData.name,
        description: newKeyData.description,
        access_level: newKeyData.access_level,
      };
      
      console.log('Creating API key with payload:', payload);
      
      const response = await fetch('/api/api-keys', {
        method: 'POST',
        headers: addCSRFHeader({
          'Content-Type': 'application/json',
          Authorization: `Bearer ${accessToken}`,
          'X-Workspace-ID': workspaceId,
        }),
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const error = await response.json();
        throw error;
      }

      const data = await response.json();
      setCreatedKey({ id: data.id, key: data.key });
      
      // Refresh the list
      await loadApiKeys();
      
      // Reset form
      setNewKeyData({ name: '', description: '', access_level: 'write' });
    } catch (error) {
      logError('Failed to create API key', error);
      toast({
        title: 'Error',
        description: 'Failed to create API key. Please try again.',
        variant: 'destructive',
      });
    } finally {
      setCreating(false);
    }
  };

  const handleDeleteKey = async () => {
    if (!selectedKey) return;

    setDeleting(selectedKey.id);
    try {
      const response = await fetch(`/api/api-keys/${selectedKey.id}`, {
        method: 'DELETE',
        headers: addCSRFHeader({
          Authorization: `Bearer ${accessToken}`,
          'X-Workspace-ID': workspaceId,
        }),
      });

      if (!response.ok) {
        const error = await response.json();
        throw error;
      }

      toast({
        title: 'Success',
        description: 'API key deleted successfully.',
      });

      // Refresh the list
      await loadApiKeys();
      
      // Close dialog
      setShowDeleteDialog(false);
      setSelectedKey(null);
    } catch (error) {
      logError('Failed to delete API key', error);
      toast({
        title: 'Error',
        description: 'Failed to delete API key. Please try again.',
        variant: 'destructive',
      });
    } finally {
      setDeleting(null);
    }
  };

  const copyToClipboard = async (text: string, keyId: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedKeyId(keyId);
      toast({
        title: 'Copied!',
        description: 'API key copied to clipboard.',
      });
      
      // Reset after 2 seconds
      setTimeout(() => setCopiedKeyId(null), 2000);
    } catch (error) {
      toast({
        title: 'Error',
        description: 'Failed to copy to clipboard.',
        variant: 'destructive',
      });
    }
  };


  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>API Keys</CardTitle>
              <CardDescription>
                Manage API keys for programmatic access to your workspace
              </CardDescription>
            </div>
            <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
              <DialogTrigger asChild>
                <Button>
                  <Plus className="mr-2 h-4 w-4" />
                  Create API Key
                </Button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-[500px]">
                <DialogHeader>
                  <DialogTitle>Create New API Key</DialogTitle>
                  <DialogDescription>
                    Create a new API key for accessing the Cyphera API
                  </DialogDescription>
                </DialogHeader>
                
                {!createdKey ? (
                  <>
                    <div className="space-y-4 py-4">
                      <div className="space-y-2">
                        <Label htmlFor="name">Name</Label>
                        <Input
                          id="name"
                          placeholder="e.g., Production Server"
                          value={newKeyData.name}
                          onChange={(e) => setNewKeyData({ ...newKeyData, name: e.target.value })}
                        />
                      </div>
                      
                      <div className="space-y-2">
                        <Label htmlFor="description">Description (optional)</Label>
                        <Input
                          id="description"
                          placeholder="e.g., Used for production API calls"
                          value={newKeyData.description}
                          onChange={(e) => setNewKeyData({ ...newKeyData, description: e.target.value })}
                        />
                      </div>
                      
                    </div>
                    
                    <DialogFooter>
                      <Button
                        variant="outline"
                        onClick={() => {
                          setShowCreateDialog(false);
                          setNewKeyData({ name: '', description: '', access_level: 'write' });
                        }}
                      >
                        Cancel
                      </Button>
                      <Button onClick={handleCreateKey} disabled={creating}>
                        {creating ? 'Creating...' : 'Create Key'}
                      </Button>
                    </DialogFooter>
                  </>
                ) : (
                  <>
                    <Alert className="my-4">
                      <AlertTriangle className="h-4 w-4" />
                      <AlertTitle>Important!</AlertTitle>
                      <AlertDescription>
                        Make sure to copy your API key now. You won't be able to see it again!
                      </AlertDescription>
                    </Alert>
                    
                    <div className="space-y-4 py-4">
                      <div className="space-y-2">
                        <Label>Your API Key</Label>
                        <div className="flex items-center space-x-2">
                          <code className="flex-1 p-2 bg-muted rounded text-sm font-mono break-all">
                            {createdKey.key}
                          </code>
                          <Button
                            size="icon"
                            variant="outline"
                            onClick={() => copyToClipboard(createdKey.key, createdKey.id)}
                          >
                            {copiedKeyId === createdKey.id ? (
                              <Check className="h-4 w-4" />
                            ) : (
                              <Copy className="h-4 w-4" />
                            )}
                          </Button>
                        </div>
                      </div>
                    </div>
                    
                    <DialogFooter>
                      <Button
                        onClick={() => {
                          setShowCreateDialog(false);
                          setCreatedKey(null);
                        }}
                      >
                        Done
                      </Button>
                    </DialogFooter>
                  </>
                )}
              </DialogContent>
            </Dialog>
          </div>
        </CardHeader>
        
        <CardContent>
          {loading ? (
            <div className="flex items-center justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
            </div>
          ) : apiKeys.length === 0 ? (
            <div className="text-center py-8">
              <Key className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <p className="text-muted-foreground">No API keys yet</p>
              <p className="text-sm text-muted-foreground mt-1">
                Create your first API key to get started
              </p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Key Prefix</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Last Used</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {apiKeys.map((key) => (
                  <TableRow key={key.id}>
                    <TableCell className="font-medium">{key.name}</TableCell>
                    <TableCell>
                      <code className="text-sm">{key.key_prefix}</code>
                    </TableCell>
                    <TableCell>
                      {formatDistanceToNow(new Date(key.created_at), { addSuffix: true })}
                    </TableCell>
                    <TableCell>
                      {key.last_used_at
                        ? formatDistanceToNow(new Date(key.last_used_at), { addSuffix: true })
                        : 'Never'}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => {
                          setSelectedKey(key);
                          setShowDeleteDialog(true);
                        }}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Delete Confirmation Dialog */}
      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete API Key</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete the API key "{selectedKey?.name}"? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setShowDeleteDialog(false);
                setSelectedKey(null);
              }}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDeleteKey}
              disabled={!!deleting}
            >
              {deleting ? 'Deleting...' : 'Delete'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}