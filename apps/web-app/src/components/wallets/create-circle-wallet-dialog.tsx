'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { useRouter } from 'next/navigation';
import { v4 as uuidv4 } from 'uuid';
import { useCircleSDK } from '@/hooks/web3';
import { toast } from 'sonner';
import { Loader2, Plus, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { NetworkWithTokensResponse } from '@/types/network';
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
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { CircleUserData, CreateWalletsRequest } from '@/types/circle';
import { logger } from '@/lib/core/logger/logger-utils';
// Zod schema for validating Circle user response
const circleUserDataSchema = z.object({
  createDate: z.string(),
  id: z.string(),
  pinDetails: z.object({
    failedAttempts: z.number(),
    lastLockOverrideDate: z.string(),
    lockedDate: z.string(),
    lockedExpiryDate: z.string(),
  }),
  pinStatus: z.string(),
  securityQuestionDetails: z.object({
    failedAttempts: z.number(),
    lastLockOverrideDate: z.string(),
    lockedDate: z.string(),
    lockedExpiryDate: z.string(),
  }),
  securityQuestionStatus: z.string(),
  status: z.string(),
});

// Form validation schema
const formSchema = z.object({
  name: z
    .string()
    .min(1, 'Wallet name is required')
    .max(30, 'Wallet name must be 30 characters or less'),
});

type FormValues = z.infer<typeof formSchema>;

interface CreateCircleWalletDialogProps {
  networks: NetworkWithTokensResponse[];
  /**
   * Optional callback when a wallet is created
   */
  onWalletCreated?: () => Promise<void>;
  /**
   * Whether the dialog is open (controlled component)
   */
  isOpen?: boolean;
  /**
   * Handle open state changes (controlled component)
   */
  onOpenChange?: (open: boolean) => void;
}

export function CreateCircleWalletDialog({
  networks,
  onWalletCreated,
  isOpen: controlledIsOpen,
  onOpenChange,
}: CreateCircleWalletDialogProps) {
  const [internalIsOpen, setInternalIsOpen] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [isCreatingUser, setIsCreatingUser] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [pendingFormData, setPendingFormData] = useState<FormValues | null>(null);
  const [savedUserToken, setSavedUserToken] = useState<string>('');
  const [validatedUserData, setValidatedUserData] = useState<CircleUserData | null>(null);
  const { client, isAuthenticated, authenticateUser } = useCircleSDK();
  const router = useRouter();

  // Determine if we're using controlled or uncontrolled state
  const isControlled = controlledIsOpen !== undefined && onOpenChange !== undefined;
  const isOpen = isControlled ? controlledIsOpen : internalIsOpen;

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: '',
    },
  });

  // Reset form and error state when dialog opens/closes
  const handleOpenChange = (open: boolean) => {
    if (!open) {
      form.reset();
      setErrorMessage(null);
    }

    if (isControlled) {
      onOpenChange(open);
    } else {
      setInternalIsOpen(open);
    }
  };

  const handleCreateWallet = async (data: FormValues) => {
    if (!client) {
      toast.error('Circle SDK is not initialized');
      return;
    }


    try {
      setIsCreating(true);
      setErrorMessage(null);

      // Store the validated user response
      let validatedUserToken = '';

      // First, create a Circle user with PIN authentication if not already authenticated
      if (!isAuthenticated) {
        setIsCreatingUser(true);

        // // Generate a unique external user ID
        const externalUserId = uuidv4();

        try {
          // First fetch CSRF token
          const csrfResponse = await fetch('/api/auth/csrf');
          if (!csrfResponse.ok) {
            throw new Error('Failed to fetch CSRF token');
          }
          const { csrfToken } = await csrfResponse.json();

          // Create the Circle user through our API endpoint
          const response = await fetch(`/api/circle/users`, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'X-CSRF-Token': csrfToken,
            },
            body: JSON.stringify({ external_user_id: externalUserId }),
          });

          if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Failed to create circle user 2');
          }

          const userResponse = await response.json();

          // Validate the user response using the schema
          try {
            const validatedResponse = circleUserDataSchema.parse(userResponse) as CircleUserData;

            // Store the validated user data
            setValidatedUserData(validatedResponse);

            // Step 2: Request a user token using the user ID
            const circleUserId = validatedResponse.id;

            const tokenResponse = await fetch(`/api/circle/users/${circleUserId}/token`, {
              method: 'POST',
              headers: {
                'Content-Type': 'application/json',
              },
            });

            if (!tokenResponse.ok) {
              const tokenErrorData = await tokenResponse.json();
              throw new Error(tokenErrorData.error || 'Failed to create user token');
            }

            const tokenData = await tokenResponse.json();

            // Store the token for later use
            validatedUserToken = tokenData.data.userToken;
            const encryptionKey = tokenData.data.encryptionKey;

            // Initialize the Circle SDK with the user token
            await authenticateUser(validatedUserToken, encryptionKey);

            // Save the token for later use in wallet creation
            setSavedUserToken(validatedUserToken);

            // Check PIN status and handle accordingly
            if (validatedResponse.pinStatus === 'LOCKED') {
              toast.error('Your PIN is currently locked. Please try again later.');
              handleOpenChange(false);
              setIsCreating(false);
              setIsCreatingUser(false);
              return;
            }

            // Store the form data for after PIN setup
            setPendingFormData(data);

            if (validatedResponse.pinStatus === 'UNSET') {
              // For users with UNSET PIN, we need to use the PIN with wallets flow
              // Create the PIN with wallets challenge
              const pinWithWalletsResponse = await fetch(`/api/circle/users/pin/create-with-wallets`, {
                method: 'POST',
                headers: {
                  'Content-Type': 'application/json',
                  'X-CSRF-Token': csrfToken,
                },
                body: JSON.stringify({
                  // Backend will auto-select blockchains if not provided
                  account_type: 'SCA',
                }),
              });

              if (!pinWithWalletsResponse.ok) {
                const errorData = await pinWithWalletsResponse.json();
                throw new Error(errorData.error || 'Failed to create PIN with wallets');
              }

              const pinWithWalletsData = await pinWithWalletsResponse.json();
              const pinChallengeId = pinWithWalletsData.challenge_id;

              if (!pinChallengeId) {
                throw new Error('Failed to get challenge ID for PIN with wallets');
              }

              // Execute the PIN with wallets challenge
              client.execute(pinChallengeId, (error) => {
                setIsCreating(false);
                setIsCreatingUser(false);
                
                if (error) {
                  setErrorMessage('Failed to complete PIN setup with wallets. Please try again.');
                  return;
                }

                // PIN setup and wallet creation completed successfully
                toast.success('Your PIN has been set up and wallets created successfully!');
                handleOpenChange(false);
                form.reset();
                onWalletCreated?.();
                router.refresh();
              });

              // Stop here - the challenge execution will handle everything
              return;
            }

            // If PIN is already enabled, continue with wallet creation
            if (validatedResponse.pinStatus === 'ENABLED') {
              await createWalletAfterPinSetup(data, validatedUserToken);
              return;
            }

            // Handle unexpected PIN status
            throw new Error(`Unexpected PIN status: ${validatedResponse.pinStatus}`);
          } catch (validationError) {
            logger.error('Invalid Circle user response:', validationError);
            throw new Error('Invalid user response format from API');
          }
        } catch (userError) {
          logger.error('Error creating Circle user:', userError);
          const userErrorMsg =
            userError instanceof Error ? userError.message : 'Failed to create Circle user 3';
          setErrorMessage(userErrorMsg);
          throw new Error(userErrorMsg);
        } finally {
          setIsCreatingUser(false);
        }
      } else {
        // Handle case where user is already authenticated but perhaps token is missing in state?
        // Need to ensure savedUserToken is valid/available before proceeding
        if (!savedUserToken) {
          // This might require re-fetching token or showing an error
          logger.error('User authenticated but token missing in state.');
          setErrorMessage('Session token issue. Please close and reopen.');
          setIsCreating(false);
          return;
        }
        validatedUserToken = savedUserToken;
        // If user is authenticated, assume PIN is handled or check validatedUserData again?
        // For simplicity, assume if authenticated, proceed (PIN check happened on first auth)
        await createWalletAfterPinSetup(data, validatedUserToken);
      }
    } catch (error) {
      logger.error('Error creating wallet:', error);
      const errorMsg =
        error instanceof Error ? error.message : 'There was an error creating your wallet';
      setErrorMessage(errorMsg);
      toast.error(errorMsg);
    } finally {
      setIsCreating(false);
    }
  };

  // Separate function to create wallet after PIN setup
  const createWalletAfterPinSetup = async (data: FormValues, userToken: string) => {
    try {
      setIsCreating(true);
      setErrorMessage(null);

      // Create the request payload - backend will handle blockchain selection
      const idempotencyKey = uuidv4();
      const requestBody: CreateWalletsRequest = {
        account_type: 'SCA',
        idempotency_key: idempotencyKey,
        user_token: userToken, // Pass the user token
        metadata: [
          {
            name: data.name,
            ref_id: idempotencyKey,
          },
        ],
      };

      // 5. Make the API call
      const walletResponse = await fetch('/api/circle/wallets', {
        // Ensure this matches your actual backend endpoint
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestBody),
      });

      if (!walletResponse.ok) {
        const errorData = await walletResponse.json().catch(() => ({})); // Add catch for non-JSON errors
        const correlationId = errorData.correlation_id || walletResponse.headers.get('X-Correlation-ID');
        
        // Log error with correlation ID for debugging
        if (correlationId) {
          console.error('Wallet creation failed', {
            correlationId,
            status: walletResponse.status,
            error: errorData.error,
          });
        }
        
        throw new Error(
          errorData.error ||
            `Failed to create wallet via backend (Status: ${walletResponse.status})`
        );
      }

      const createResponse = await walletResponse.json();

      if (createResponse && createResponse.challenge_id) {
        // Execute the challenge to complete wallet creation
        await new Promise<void>((resolve, reject) => {
          if (!client) {
            reject(new Error('Circle SDK client is not initialized'));
            return;
          }

          client.execute(createResponse.challenge_id, (error) => {
            if (error) {
              logger.error('Wallet creation failed:', error);
              reject(error);
            } else {
              resolve();
            }
          });
        });

        toast.success(`${data.name} wallet created successfully`);
        handleOpenChange(false);
        form.reset();
        await onWalletCreated?.();
        router.refresh();
      } else {
        // This path might indicate the backend structure is different or an error occurred before challenge
        logger.warn('Wallet creation response did not contain a challenge ID.', createResponse);
        // Assume success if no challenge needed? Or throw error?
        // For now, let's assume success if ok and no challenge
        toast.success(`${data.name} wallet created`);
        handleOpenChange(false);
        form.reset();
        await onWalletCreated?.();
        router.refresh();
        // throw new Error('Failed to create wallet challenge');
      }
    } catch (error) {
      logger.error('Error creating wallet:', error);
      // Display specific error message from the catch block
      setErrorMessage(error instanceof Error ? error.message : 'Failed to create wallet');
      // Throw error to be caught by the caller if needed, or just show message
      // throw error;
    } finally {
      setIsCreating(false);
    }
  };


  return (
    <>
      <Dialog open={isOpen} onOpenChange={handleOpenChange}>
        {/* Only render DialogTrigger when not controlled externally */}
        {!isControlled && (
          <DialogTrigger asChild>
            <Button className="flex items-center gap-2">
              <Plus className="h-4 w-4" />
              Create Circle Wallet
            </Button>
          </DialogTrigger>
        )}
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>Create Circle Wallet</DialogTitle>
            <DialogDescription>
              Create a new user-controlled wallet. The wallet will be created on all supported networks.
            </DialogDescription>
          </DialogHeader>

          {errorMessage && (
            <Alert variant="destructive" className="mt-2">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{errorMessage}</AlertDescription>
            </Alert>
          )}

          <Form {...form}>
            <form onSubmit={form.handleSubmit(handleCreateWallet)} className="space-y-4">
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Wallet Nickname</FormLabel>
                    <FormControl>
                      <Input placeholder="My Circle Wallet" {...field} />
                    </FormControl>
                    <FormDescription>A friendly name for your wallet</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />


              <DialogFooter>
                <Button
                  type="submit"
                  disabled={isCreating || isCreatingUser}
                >
                  {isCreatingUser ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Creating User...
                    </>
                  ) : isCreating ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Creating Wallet...
                    </>
                  ) : (
                    'Create Wallet'
                  )}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>
    </>
  );
}
