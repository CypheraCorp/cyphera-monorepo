'use client';

import { useState, useEffect } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { Button } from '@/components/ui/button';
import { useCreateProduct, useCreateWallet } from '@/hooks/data';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormDescription,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useToast } from '@/components/ui/use-toast';
import { PRICE_TYPES, INTERVAL_TYPES } from '@/lib/constants/products';

import type { WalletResponse } from '@/types/wallet';
import { CreateProductRequest } from '@/types/product';
import { Badge } from '@/components/ui/badge';
import { NetworkWithTokensResponse } from '@/types/network';
import { PricingPreviewCard } from '@/components/ui/pricing-preview-card';
import { StepNavigation } from '@/components/ui/step-navigation';
import { CurrencyPriceInput } from '@/components/ui/currency-price-input';
import { NetworkTokenSelector } from '@/components/ui/network-token-selector';
import { WalletCardSelector } from '@/components/ui/wallet-card-selector';
import { CreateWalletInlineForm } from '@/components/wallets/create-wallet-inline-form';
import { Package, DollarSign, CreditCard, Wallet, CheckCircle, Loader2 } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { logger } from '@/lib/core/logger/logger-utils';

interface CreateProductMultiStepDialogProps {
  trigger: React.ReactNode;
  networks: NetworkWithTokensResponse[];
  wallets: WalletResponse[];
  supportedCurrencies: readonly string[];
  defaultCurrency: string;
}

// Define the steps for our multi-step form
const FORM_STEPS = [
  {
    id: 'basics',
    title: 'Product Basics',
    icon: <Package className="w-4 h-4" />,
    description: 'Name and description',
  },
  {
    id: 'pricing',
    title: 'Pricing & Billing',
    icon: <DollarSign className="w-4 h-4" />,
    description: 'Set your price and billing',
  },
  {
    id: 'payment-methods',
    title: 'Payment Options',
    icon: <CreditCard className="w-4 h-4" />,
    description: 'Choose accepted tokens',
  },
  {
    id: 'payout',
    title: 'Payout Setup',
    icon: <Wallet className="w-4 h-4" />,
    description: 'Where you receive payments',
  },
  {
    id: 'review',
    title: 'Review & Create',
    icon: <CheckCircle className="w-4 h-4" />,
    description: 'Confirm and create',
  },
];

const productFormSchema = z
  .object({
    name: z.string().min(1, 'Name is required'),
    wallet_id: z.string().optional(),
    new_wallet_address: z
      .string()
      .optional()
      .refine((val) => !val || /^0x[a-fA-F0-9]{40}$/.test(val), {
        message: 'Invalid address format. Must start with 0x followed by 40 hex characters.',
      })
      .transform((val) => (val ? val.toLowerCase() : undefined)),
    new_wallet_network_id: z.string().optional(),
    description: z.string().optional(),
    image_url: z.string().url('Invalid URL').optional().or(z.literal('')),
    url: z.string().url('Invalid URL').optional().or(z.literal('')),
    metadata: z.record(z.unknown()).optional(),
    product_tokens: z
      .array(
        z.object({
          network_id: z.string(),
          token_id: z.string(),
          active: z.boolean(),
        })
      )
      .min(1, 'At least one payment option is required'),
    active: z.boolean(),

    priceDetails: z.object({
      type: z.enum([PRICE_TYPES.RECURRING, PRICE_TYPES.ONE_TIME]),
      nickname: z.string().optional(),
      currency: z.string().min(1, 'Currency is required'),
      unit_amount_in_pennies: z.coerce.number().min(0, 'Price must be >= 0'),
      interval_type: z
        .enum([
          INTERVAL_TYPES.ONE_MINUTE,
          INTERVAL_TYPES.FIVE_MINUTES,
          INTERVAL_TYPES.DAILY,
          INTERVAL_TYPES.WEEKLY,
          INTERVAL_TYPES.MONTHLY,
          INTERVAL_TYPES.YEARLY,
        ])
        .optional(),
      interval_count: z.coerce.number().positive().optional(),
      term_length: z.coerce.number().positive().optional(),
      active: z.boolean(),
    }),
  })
  .refine(
    (data) => {
      if (data.priceDetails.type === PRICE_TYPES.RECURRING) {
        return !!data.priceDetails.interval_type && !!data.priceDetails.term_length;
      }
      return true;
    },
    {
      message: '', // Silent validation - prevents form submission but no error message shown
      path: ['priceDetails'],
    }
  )
  .superRefine((data, ctx) => {
    const hasExistingWallet = !!data.wallet_id;
    const hasNewWalletAddress = !!data.new_wallet_address;
    const hasNewWalletNetwork = !!data.new_wallet_network_id;

    if (!hasExistingWallet && !hasNewWalletAddress) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: 'Select an existing wallet or enter a new wallet address.',
        path: ['wallet_id'],
      });
    }
    if (hasExistingWallet && hasNewWalletAddress) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: 'Cannot select an existing wallet and add a new one.',
        path: ['new_wallet_address'],
      });
    }
    if (hasNewWalletAddress && !hasNewWalletNetwork) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: 'Network selection is required for a new wallet address.',
        path: ['new_wallet_network_id'],
      });
    }
  });

type ProductFormValues = z.infer<typeof productFormSchema>;

export function CreateProductMultiStepDialog({
  trigger,
  networks,
  wallets,
  supportedCurrencies,
  defaultCurrency,
}: CreateProductMultiStepDialogProps) {
  const [open, setOpen] = useState(false);
  const [currentStep, setCurrentStep] = useState(0);
  const [completedSteps, setCompletedSteps] = useState<number[]>([]);
  const { toast } = useToast();

  // React Query mutations
  const createProductMutation = useCreateProduct();
  const createWalletMutation = useCreateWallet();
  const isSubmitting = createProductMutation.isPending || createWalletMutation.isPending;

  const form = useForm<ProductFormValues>({
    resolver: zodResolver(productFormSchema),
    mode: 'onChange',
    defaultValues: {
      name: '',
      description: '',
      active: true,
      product_tokens: [],
      wallet_id: undefined,
      new_wallet_address: undefined,
      new_wallet_network_id: undefined,
      image_url: '',
      url: '',
      priceDetails: {
        type: PRICE_TYPES.RECURRING,
        nickname: '',
        currency: defaultCurrency,
        unit_amount_in_pennies: 0,
        interval_type: INTERVAL_TYPES.MONTHLY,
        term_length: undefined,
        active: true,
      },
    },
  });

  // Step validation functions
  const validateStep = async (stepIndex: number): Promise<boolean> => {
    switch (stepIndex) {
      case 0: // Product Basics
        return await form.trigger(['name', 'description']);
      case 1: // Pricing & Billing
        // Manually touch the price fields before validation
        form.setValue('priceDetails.currency', form.getValues('priceDetails.currency'), {
          shouldTouch: true,
        });
        form.setValue(
          'priceDetails.unit_amount_in_pennies',
          form.getValues('priceDetails.unit_amount_in_pennies'),
          { shouldTouch: true }
        );
        if (form.getValues('priceDetails.type') === PRICE_TYPES.RECURRING) {
          form.setValue(
            'priceDetails.interval_type',
            form.getValues('priceDetails.interval_type'),
            { shouldTouch: true }
          );
          form.setValue('priceDetails.term_length', form.getValues('priceDetails.term_length'), {
            shouldTouch: true,
          });
        }
        return await form.trigger(['priceDetails']);
      case 2: // Payment Methods
        return await form.trigger(['product_tokens']);
      case 3: // Payout Setup
        return await form.trigger(['wallet_id', 'new_wallet_address', 'new_wallet_network_id']);
      case 4: // Review
        return await form.trigger();
      default:
        return true;
    }
  };

  const handleNextStep = async () => {
    const isCurrentStepValid = await validateStep(currentStep);

    if (isCurrentStepValid) {
      setCompletedSteps((prev) => [...prev.filter((step) => step !== currentStep), currentStep]);
      if (currentStep < FORM_STEPS.length - 1) {
        setCurrentStep(currentStep + 1);
      }
    }
  };

  const handlePrevStep = () => {
    if (currentStep > 0) {
      setCurrentStep(currentStep - 1);
    }
  };

  const handleStepClick = (stepIndex: number) => {
    // Allow navigation to completed steps or the next immediate step
    if (completedSteps.includes(stepIndex) || stepIndex <= Math.max(...completedSteps, -1) + 1) {
      setCurrentStep(stepIndex);
    }
  };

  // Create wallet function
  async function createWallet(address: string, networkId: string): Promise<string> {
    try {
      const wallet = await createWalletMutation.mutateAsync({
        wallet_address: address,
        network_id: networkId,
        is_primary: false,
        verified: false,
      });

      toast({
        title: 'New Wallet Created',
        description: `Wallet ${address.substring(0, 6)}... added.`,
      });

      return wallet.id;
    } catch (error) {
      logger.error('Failed to create wallet:', error);
      toast({
        title: 'Wallet Creation Error',
        description: error instanceof Error ? error.message : 'Unknown error',
        variant: 'destructive',
      });
      throw error;
    }
  }

  // Submit function (same logic as original)
  async function onSubmit(data: ProductFormValues) {
    let finalWalletId = data.wallet_id;

    try {
      if (data.new_wallet_address && data.new_wallet_network_id) {
        finalWalletId = await createWallet(data.new_wallet_address, data.new_wallet_network_id!);
      } else if (!finalWalletId) {
        toast({
          title: 'Validation Error',
          description: 'Please select or add a wallet.',
          variant: 'destructive',
        });
        return;
      }

      const walletNetworkId =
        data.new_wallet_network_id || wallets.find((w) => w.id === finalWalletId)?.network_id;

      if (!walletNetworkId) {
        toast({
          title: 'Wallet Error',
          description: 'Could not determine the network for the selected payout wallet.',
          variant: 'destructive',
        });
        return;
      }

      const productTokenNetworkIds = new Set(
        data.product_tokens.map(
          (token: { network_id: string; token_id: string; active: boolean }) => token.network_id
        )
      );

      if (!productTokenNetworkIds.has(walletNetworkId)) {
        toast({
          title: 'Network Mismatch',
          description: 'The payout wallet must be on a network selected in the Payment Options.',
          variant: 'destructive',
        });
        return;
      }

      const productTokensPayload = data.product_tokens.map(
        (token: { network_id: string; token_id: string; active: boolean }) => ({
          network_id: token.network_id,
          token_id: token.token_id,
          active: token.active,
          // product_id will be filled by the backend
        })
      );

      // Create product with embedded price fields (prices table merged into products)
      const finalProductData: CreateProductRequest = {
        name: data.name,
        description: data.description || undefined,
        active: data.active,
        wallet_id: finalWalletId!,
        product_tokens: productTokensPayload,
        image_url: data.image_url || undefined,
        url: data.url || undefined,
        metadata: data.metadata || undefined,
        // Embedded price fields (now part of product)
        price_type: data.priceDetails.type,
        currency: data.priceDetails.currency,
        unit_amount_in_pennies: data.priceDetails.unit_amount_in_pennies,
        interval_type:
          data.priceDetails.type === PRICE_TYPES.RECURRING
            ? data.priceDetails.interval_type!
            : undefined,
        term_length:
          data.priceDetails.type === PRICE_TYPES.RECURRING
            ? data.priceDetails.term_length!
            : undefined,
        price_nickname: data.priceDetails.nickname || undefined,
      };

      await createProductMutation.mutateAsync(finalProductData);

      toast({ title: 'Success', description: 'Product created successfully' });
      setOpen(false);
      form.reset();
      setCurrentStep(0);
      setCompletedSteps([]);
    } catch (error) {
      logger.error('Failed to create product:', error);
      if (!(error instanceof Error && error.message.includes('Failed to create wallet'))) {
        toast({
          title: 'Error',
          description: error instanceof Error ? error.message : 'Please try again.',
          variant: 'destructive',
        });
      }
    }
  }

  // Reset form when dialog closes
  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen) {
      form.reset();
      setCurrentStep(0);
      setCompletedSteps([]);
    }
    setOpen(newOpen);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="sm:max-w-[900px] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Create Product</DialogTitle>
          <DialogDescription>
            Let&apos;s create your product step by step. This makes it easier and ensures everything
            is set up correctly.
          </DialogDescription>
        </DialogHeader>

        <StepNavigation
          steps={FORM_STEPS}
          currentStep={currentStep}
          completedSteps={completedSteps}
          onStepClick={handleStepClick}
          className="mb-6"
        />

        <Form {...form}>
          <form className="space-y-6">
            {/* Step Content */}
            <div className="min-h-[400px]">
              {currentStep === 0 && <ProductBasicsStep form={form} />}
              {currentStep === 1 && (
                <PricingBillingStep
                  form={form}
                  supportedCurrencies={supportedCurrencies}
                  networks={networks}
                />
              )}
              {currentStep === 2 && <PaymentMethodsStep form={form} networks={networks} />}
              {currentStep === 3 && <PayoutSetupStep form={form} wallets={wallets} />}
              {currentStep === 4 && (
                <ReviewCreateStep
                  form={form}
                  networks={networks}
                  wallets={wallets}
                  onConfirmCreate={() => form.handleSubmit(onSubmit)()}
                  isCreating={isSubmitting}
                />
              )}
            </div>

            {/* Navigation Buttons */}
            <div className="flex justify-between pt-6 border-t">
              <Button
                type="button"
                variant="outline"
                onClick={handlePrevStep}
                disabled={currentStep === 0 || isSubmitting}
              >
                Previous
              </Button>

              <div className="flex gap-2">
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => handleOpenChange(false)}
                  disabled={isSubmitting}
                >
                  Cancel
                </Button>

                {currentStep < FORM_STEPS.length - 1 && (
                  <Button type="button" onClick={handleNextStep} disabled={isSubmitting}>
                    Continue
                  </Button>
                )}
              </div>
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

// Step Components (simplified for now - we'll enhance these in later phases)
function ProductBasicsStep({ form }: { form: ReturnType<typeof useForm<ProductFormValues>> }) {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-semibold mb-2">Tell us about your product</h3>
        <p className="text-sm text-muted-foreground mb-4">
          Start with the basics - what are you selling and how would you describe it?
        </p>
      </div>

      <FormField
        control={form.control}
        name="name"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Product Name *</FormLabel>
            <FormControl>
              <Input
                placeholder="e.g., Premium Newsletter, Pro Software License"
                {...field}
                className="text-lg"
              />
            </FormControl>
          </FormItem>
        )}
      />

      <FormField
        control={form.control}
        name="description"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Description</FormLabel>
            <FormControl>
              <Textarea
                placeholder="Describe what customers get with this product..."
                {...field}
                value={field.value || ''}
                rows={4}
              />
            </FormControl>
            <FormDescription>Help customers understand the value of your product</FormDescription>
          </FormItem>
        )}
      />
    </div>
  );
}

function PricingBillingStep({
  form,
  supportedCurrencies,
  networks,
}: {
  form: ReturnType<typeof useForm<ProductFormValues>>;
  supportedCurrencies: readonly string[];
  networks: NetworkWithTokensResponse[];
}) {
  const productType = form.watch('priceDetails.type');
  const productName = form.watch('name');
  const productDescription = form.watch('description');
  const priceInPennies = form.watch('priceDetails.unit_amount_in_pennies');
  const currency = form.watch('priceDetails.currency');
  const intervalType = form.watch('priceDetails.interval_type');
  const termLength = form.watch('priceDetails.term_length');
  const productTokens = form.watch('product_tokens');

  // Extract selected network and token info for preview (show first option)
  const selectedToken = productTokens?.[0];
  const selectedNetwork = selectedToken
    ? networks.find((n) => n.network.id === selectedToken.network_id)
    : null;
  const selectedTokenInfo =
    selectedNetwork && selectedToken
      ? selectedNetwork.tokens.find((t) => t.id === selectedToken.token_id)
      : null;

  // If multiple payment options, show count
  const totalPaymentOptions = productTokens?.length || 0;

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
      {/* Left Column - Form Fields */}
      <div className="space-y-6">
        <div>
          <h3 className="text-lg font-semibold mb-2">Set your pricing</h3>
          <p className="text-sm text-muted-foreground mb-4">
            Choose how customers will pay for your product
          </p>
        </div>

        <FormField
          control={form.control}
          name="priceDetails.type"
          render={({ field }) => (
            <FormItem>
              <div className="space-y-2">
                <label
                  htmlFor={field.name}
                  className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                >
                  Product Type *
                </label>
                <Select onValueChange={field.onChange} value={field.value}>
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue placeholder="Select type" />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    <SelectItem value={PRICE_TYPES.ONE_TIME}>One-Time Purchase</SelectItem>
                    <SelectItem value={PRICE_TYPES.RECURRING}>Subscription</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </FormItem>
          )}
        />

        {/* Currency and Price Input */}
        <FormField
          control={form.control}
          name="priceDetails"
          render={() => (
            <FormItem>
              <FormLabel>Price *</FormLabel>
              <FormControl>
                <CurrencyPriceInput
                  currency={currency}
                  onCurrencyChange={(currency) => form.setValue('priceDetails.currency', currency)}
                  priceInPennies={priceInPennies}
                  onPriceChange={(pennies) =>
                    form.setValue('priceDetails.unit_amount_in_pennies', pennies)
                  }
                  supportedCurrencies={supportedCurrencies}
                  placeholder="19.99"
                />
              </FormControl>
              {/* Show form errors for price fields - only after field is touched */}
              <div>
                {form.formState.touchedFields?.priceDetails?.currency &&
                  form.formState.errors?.priceDetails?.currency && (
                    <p className="text-sm font-medium text-destructive">
                      {form.formState.errors.priceDetails.currency.message}
                    </p>
                  )}
                {form.formState.touchedFields?.priceDetails?.unit_amount_in_pennies &&
                  form.formState.errors?.priceDetails?.unit_amount_in_pennies && (
                    <p className="text-sm font-medium text-destructive">
                      {form.formState.errors.priceDetails.unit_amount_in_pennies.message}
                    </p>
                  )}
              </div>
            </FormItem>
          )}
        />

        {productType === PRICE_TYPES.RECURRING && (
          <div className="grid grid-cols-2 gap-4">
            <FormField
              control={form.control}
              name="priceDetails.interval_type"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Billing Interval *</FormLabel>
                  <Select
                    onValueChange={field.onChange}
                    value={field.value || INTERVAL_TYPES.MONTHLY}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select interval" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value={INTERVAL_TYPES.ONE_MINUTE}>1 Minute (Dev)</SelectItem>
                      <SelectItem value={INTERVAL_TYPES.FIVE_MINUTES}>5 Minutes (Dev)</SelectItem>
                      <SelectItem value={INTERVAL_TYPES.DAILY}>Daily</SelectItem>
                      <SelectItem value={INTERVAL_TYPES.WEEKLY}>Weekly</SelectItem>
                      <SelectItem value={INTERVAL_TYPES.MONTHLY}>Monthly</SelectItem>
                      <SelectItem value={INTERVAL_TYPES.YEARLY}>Yearly</SelectItem>
                    </SelectContent>
                  </Select>
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="priceDetails.term_length"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Term Length *</FormLabel>
                  <FormControl>
                    <Input
                      type="text"
                      inputMode="numeric"
                      placeholder="12"
                      {...field}
                      value={field.value ?? ''}
                      onChange={(e) => {
                        const value = e.target.value;

                        // Allow empty value
                        if (value === '') {
                          field.onChange(undefined);
                          return;
                        }

                        // Only allow positive whole numbers (digits only)
                        const digitsOnly = value.replace(/[^0-9]/g, '');

                        if (digitsOnly === '') {
                          field.onChange(undefined);
                        } else {
                          const numValue = parseInt(digitsOnly, 10);
                          // Prevent leading zeros by setting the display value
                          e.target.value = numValue.toString();
                          field.onChange(numValue);
                        }
                      }}
                    />
                  </FormControl>
                  <FormDescription>Number of billing intervals (e.g., 12 months)</FormDescription>
                </FormItem>
              )}
            />
          </div>
        )}
      </div>

      {/* Right Column - Live Preview */}
      <div className="space-y-4">
        <div>
          <h4 className="text-sm font-medium text-muted-foreground mb-3">Live Preview</h4>
          <PricingPreviewCard
            productName={productName}
            productDescription={productDescription}
            priceInPennies={priceInPennies}
            currency={currency}
            productType={productType}
            intervalType={intervalType}
            termLength={termLength}
            selectedNetworkName={selectedNetwork?.network.name}
            selectedTokenSymbol={selectedTokenInfo?.symbol}
            totalPaymentOptions={totalPaymentOptions}
          />
        </div>
      </div>
    </div>
  );
}

function PaymentMethodsStep({
  form,
  networks,
}: {
  form: ReturnType<typeof useForm<ProductFormValues>>;
  networks: NetworkWithTokensResponse[];
}) {
  // Initialize selected tokens from form state
  const formTokens = form.watch('product_tokens') || [];

  // Handle multiple selected tokens
  const [selectedTokens, setSelectedTokens] = useState<
    Array<{
      network_id: string;
      token_id: string;
      network_name: string;
      token_symbol: string;
      token_name: string;
      is_gas_token: boolean;
    }>
  >(() => {
    // Initialize from form state if available
    return formTokens
      .map((token: { network_id: string; token_id: string; active: boolean }) => {
        const network = networks.find((n) => n.network.id === token.network_id);
        const tokenInfo = network?.tokens.find((t) => t.id === token.token_id);
        return {
          network_id: token.network_id,
          token_id: token.token_id,
          network_name: network?.network.name || '',
          token_symbol: tokenInfo?.symbol || '',
          token_name: tokenInfo?.name || '',
          is_gas_token: tokenInfo?.gas_token || false,
        };
      })
      .filter((token) => token.network_name && token.token_symbol);
  });

  const handleTokensChange = (tokens: typeof selectedTokens) => {
    setSelectedTokens(tokens);
    // Convert to the format expected by the form
    const formattedTokens = tokens.map((token) => ({
      network_id: token.network_id,
      token_id: token.token_id,
      active: true,
    }));
    form.setValue('product_tokens', formattedTokens);
  };

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-semibold mb-2">Payment Options</h3>
        <p className="text-sm text-muted-foreground mb-4">
          First select a network, then choose which tokens your customers can use to pay on that
          network.
        </p>
      </div>

      <NetworkTokenSelector
        networks={networks}
        selectedTokens={selectedTokens}
        onTokensChange={handleTokensChange}
      />

      {/* Hidden form field for validation */}
      <div className="hidden">
        <FormField
          control={form.control}
          name="product_tokens"
          render={({ field }) => (
            <FormControl>
              <input
                {...field}
                value={JSON.stringify(field.value || [])}
                onChange={() => {}} // Read-only, value set by handleTokensChange
              />
            </FormControl>
          )}
        />
      </div>
    </div>
  );
}

function PayoutSetupStep({
  form,
  wallets,
}: {
  form: ReturnType<typeof useForm<ProductFormValues>>;
  wallets: WalletResponse[];
}) {
  const [selectedWalletId, setSelectedWalletId] = useState<string>('');
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [availableWallets, setAvailableWallets] = useState<WalletResponse[]>(wallets);

  // Sync wallets prop with local state
  useEffect(() => {
    setAvailableWallets(wallets);
  }, [wallets]);

  // Get selected payment options to filter wallets by network
  const productTokens =
    (form.watch('product_tokens') as Array<{
      network_id: string;
      token_id: string;
      active: boolean;
    }>) || [];

  // Extract network IDs from selected payment options
  const selectedNetworkIds = new Set(productTokens.map((token) => token.network_id));

  // Filter wallets to only show ones that match the selected payment networks
  const compatibleWallets = availableWallets.filter((wallet) =>
    selectedNetworkIds.has(wallet.network_id || '')
  );

  const handleWalletSelect = (walletId: string) => {
    setSelectedWalletId(walletId);
    form.setValue('wallet_id', walletId);
    // Clear new wallet fields when existing wallet is selected
    form.setValue('new_wallet_address', undefined);
    form.setValue('new_wallet_network_id', undefined);
  };

  const handleCreateWallet = () => {
    setShowCreateForm(true);
  };

  const handleWalletCreated = (wallet: WalletResponse) => {
    setSelectedWalletId(wallet.id);
    setShowCreateForm(false);
  };

  const handleCancelCreate = () => {
    setShowCreateForm(false);
  };

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-semibold mb-2">Where should payments go?</h3>
        <p className="text-sm text-muted-foreground mb-4">
          Choose the wallet where you&apos;ll receive payments from customers.
        </p>
      </div>

      {showCreateForm ? (
        <CreateWalletInlineForm
          onWalletCreated={handleWalletCreated}
          onCancel={handleCancelCreate}
        />
      ) : compatibleWallets.length === 0 && availableWallets.length > 0 ? (
        <div className="bg-orange-50 dark:bg-orange-950/50 rounded-lg p-6 border-2 border-dashed border-orange-200 dark:border-orange-800 text-center">
          <Wallet className="h-8 w-8 text-orange-500 mx-auto mb-3" />
          <h4 className="font-medium text-orange-900 dark:text-orange-100 mb-2">
            No Compatible Wallets Found
          </h4>
          <p className="text-sm text-orange-700 dark:text-orange-300 mb-4">
            You have {availableWallets.length} wallet(s), but none are on the networks you selected
            for payments ({Array.from(selectedNetworkIds).join(', ')}).
          </p>
          <p className="text-xs text-orange-600 dark:text-orange-400 mb-4">
            Create a new wallet on one of these networks or go back and adjust your payment options.
          </p>
          <Button
            variant="outline"
            onClick={handleCreateWallet}
            className="flex items-center gap-2 border-orange-300 text-orange-700 hover:bg-orange-100 dark:border-orange-700 dark:text-orange-300 dark:hover:bg-orange-900"
          >
            <Wallet className="h-4 w-4" />
            Create Compatible Wallet
          </Button>
        </div>
      ) : (
        <WalletCardSelector
          wallets={compatibleWallets}
          selectedWalletId={selectedWalletId}
          onWalletSelect={handleWalletSelect}
          onCreateWallet={handleCreateWallet}
        />
      )}

      {/* Hidden form fields for validation */}
      <div className="hidden">
        <FormField
          control={form.control}
          name="wallet_id"
          render={({ field }) => (
            <FormControl>
              <input {...field} />
            </FormControl>
          )}
        />
      </div>
    </div>
  );
}

function ReviewCreateStep({
  form,
  networks,
  wallets,
  onConfirmCreate,
  isCreating,
}: {
  form: ReturnType<typeof useForm<ProductFormValues>>;
  networks: NetworkWithTokensResponse[];
  wallets: WalletResponse[];
  onConfirmCreate: () => void;
  isCreating: boolean;
}) {
  const formData = form.watch();

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center space-y-3">
        <div className="w-12 h-12 bg-gradient-to-br from-blue-500 to-purple-600 rounded-xl flex items-center justify-center mx-auto">
          <Package className="w-6 h-6 text-white" />
        </div>
        <div>
          <h3 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
            Review your product
          </h3>
          <p className="text-gray-600 dark:text-gray-400 max-w-md mx-auto">
            Please review all details carefully. Once you confirm, your product will be created and
            ready for customers.
          </p>
        </div>
      </div>

      {/* Product Overview Card */}
      <Card className="overflow-hidden border-0 shadow-lg bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800">
        <div className="bg-gradient-to-r from-blue-600 to-purple-600 p-8">
          <div className="text-center text-white">
            <h4 className="text-2xl font-bold">{formData.name}</h4>
            {formData.description && (
              <p className="text-blue-100 mt-2 opacity-90">{formData.description}</p>
            )}
          </div>
        </div>

        <CardContent className="p-6">
          <div className="grid md:grid-cols-2 gap-6">
            {/* Accepted Payments */}
            <div className="space-y-3">
              <h5 className="font-semibold text-gray-900 dark:text-gray-100 flex items-center gap-2">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                Accepted Payments
              </h5>
              <div className="flex flex-wrap gap-2">
                {formData.product_tokens?.map(
                  (token: { network_id: string; token_id: string; active: boolean }) => {
                    const network = networks.find((n) => n.network.id === token.network_id);
                    const tokenInfo = network?.tokens.find((t) => t.id === token.token_id);

                    return (
                      <div
                        key={`${token.network_id}-${token.token_id}`}
                        className="inline-flex items-center gap-2 bg-white dark:bg-gray-800 px-3 py-1.5 rounded-full border border-gray-200 dark:border-gray-700 shadow-sm"
                      >
                        <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                        <span className="text-sm font-medium">
                          {network?.network.name} {tokenInfo?.symbol}
                        </span>
                      </div>
                    );
                  }
                )}
              </div>
            </div>

            {/* Billing Calculations */}
            {formData.priceDetails.type === PRICE_TYPES.RECURRING && (
              <div className="space-y-3">
                <h5 className="font-semibold text-gray-900 dark:text-gray-100 flex items-center gap-2">
                  <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                  Billing Details
                </h5>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-gray-600 dark:text-gray-400">Per cycle:</span>
                    <span className="font-medium">
                      {formData.priceDetails.currency}{' '}
                      {(formData.priceDetails.unit_amount_in_pennies / 100).toFixed(2)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-600 dark:text-gray-400">Frequency:</span>
                    <span className="font-medium">
                      Every {formData.priceDetails.interval_type?.replace('_', ' ')}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-600 dark:text-gray-400">Total cycles:</span>
                    <span className="font-medium">{formData.priceDetails.term_length}</span>
                  </div>
                  <div className="border-t pt-2 flex justify-between">
                    <span className="font-medium text-gray-900 dark:text-gray-100">
                      Total amount:
                    </span>
                    <span className="font-bold text-gray-900 dark:text-gray-100">
                      {formData.priceDetails.currency}{' '}
                      {(
                        (formData.priceDetails.unit_amount_in_pennies / 100) *
                        (formData.priceDetails.term_length || 1)
                      ).toFixed(2)}
                    </span>
                  </div>
                </div>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Payout Information */}
      <Card className="border-0 shadow-lg">
        <CardContent className="p-6">
          <div className="flex items-start gap-4">
            <div className="w-12 h-12 bg-gradient-to-br from-purple-500 to-pink-600 rounded-xl flex items-center justify-center flex-shrink-0">
              <Wallet className="w-6 h-6 text-white" />
            </div>
            <div className="flex-1">
              <h5 className="font-semibold text-gray-900 dark:text-gray-100 mb-2">
                Payout Destination
              </h5>
              {formData.wallet_id ? (
                <div>
                  {(() => {
                    const wallet = wallets.find((w) => w.id === formData.wallet_id);
                    const network = networks.find((n) => n.network.id === wallet?.network_id);
                    return (
                      <div className="space-y-2">
                        <div className="flex items-center gap-3">
                          <span className="font-medium text-gray-900 dark:text-gray-100">
                            {wallet?.nickname || `Wallet ${wallet?.id}`}
                          </span>
                          <Badge
                            variant="outline"
                            className="bg-purple-50 text-purple-700 border-purple-200"
                          >
                            {network?.network.name}
                          </Badge>
                        </div>
                        <p className="text-sm text-gray-600 dark:text-gray-400 font-mono">
                          {wallet?.wallet_address}
                        </p>
                      </div>
                    );
                  })()}
                </div>
              ) : (
                <div className="space-y-2">
                  <div className="flex items-center gap-3">
                    <span className="font-medium text-gray-900 dark:text-gray-100">New Wallet</span>
                    <Badge
                      variant="outline"
                      className="bg-purple-50 text-purple-700 border-purple-200"
                    >
                      {
                        networks.find((n) => n.network.id === formData.new_wallet_network_id)
                          ?.network.name
                      }
                    </Badge>
                  </div>
                  <p className="text-sm text-gray-600 dark:text-gray-400 font-mono">
                    {formData.new_wallet_address}
                  </p>
                </div>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Confirmation Section */}
      <Card className="border-0 shadow-lg">
        <CardContent className="p-6">
          <div className="text-center space-y-4">
            <div className="w-12 h-12 bg-green-600 rounded-xl flex items-center justify-center mx-auto">
              <CheckCircle className="w-6 h-6 text-white" />
            </div>

            <div>
              <h4 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                Ready to Launch?
              </h4>
              <p className="text-gray-600 dark:text-gray-400 text-sm mt-1">
                Your product will be created and available for customers.
              </p>
            </div>

            <Button
              onClick={onConfirmCreate}
              disabled={isCreating}
              className="bg-green-600 hover:bg-green-700 text-white px-6 py-2"
            >
              {isCreating ? (
                <div className="flex items-center gap-2">
                  <Loader2 className="w-4 h-4 animate-spin" />
                  Creating...
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <CheckCircle className="w-4 h-4" />
                  Create Product
                </div>
              )}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
