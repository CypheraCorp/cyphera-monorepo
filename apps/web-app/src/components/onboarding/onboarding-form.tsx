'use client';

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useRouter } from 'next/navigation';
import { useToast } from '@/components/ui/use-toast';
import { Button } from '@/components/ui/button';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  onboardingFormSchema,
  type OnboardingFormValues,
} from '@/lib/core/validation/schemas/onboarding';
import { Loader2 } from 'lucide-react';
import { useState, useEffect } from 'react';
import Script from 'next/script';
import dynamic from 'next/dynamic';
import { useEnvConfig } from '@/components/env/client';
import { logger } from '@/lib/core/logger/logger-utils';

// Dynamically import the GooglePlacesAutocomplete with no SSR
const DynamicGooglePlacesAutocomplete = dynamic(() => import('react-google-places-autocomplete'), {
  ssr: false,
});

export function OnboardingForm() {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isGoogleLoaded, setIsGoogleLoaded] = useState(false);
  const [scriptError, setScriptError] = useState<string | null>(null);

  const router = useRouter();
  const { toast } = useToast();
  const envConfig = useEnvConfig();

  // Authentication is now handled by middleware and session store
  // No need for client-side JWT management

  // Get Google Maps API key from environment config
  const googleMapsApiKey = envConfig.googleMapsApiKey;

  useEffect(() => {
    if (!googleMapsApiKey) {
      setScriptError('Google Maps API key is not configured');
      toast({
        title: 'Configuration Error',
        description:
          'Google Maps is not properly configured. Address autocomplete will not be available.',
        variant: 'destructive',
      });
    }
  }, [toast, googleMapsApiKey]);

  const form = useForm<OnboardingFormValues>({
    resolver: zodResolver(onboardingFormSchema),
    defaultValues: {
      first_name: '',
      last_name: '',
      address_line1: '',
      address_line2: '',
      city: '',
      state: '',
      country: '',
      postal_code: '',
      wallet_address: '',
    },
  });

  // Type for Google Places Autocomplete selection
  interface PlaceSelection {
    value?: {
      place_id: string;
      description?: string;
      structured_formatting?: {
        main_text?: string;
        secondary_text?: string;
      };
    };
    label?: string;
  }

  // Handle Google Places selection
  const handlePlaceSelect = async (place: PlaceSelection | null) => {
    if (!place?.value?.place_id) return;

    try {
      const geocoder = new google.maps.Geocoder();
      const result = await new Promise((resolve, reject) => {
        geocoder.geocode({ placeId: place.value?.place_id }, (results, status) => {
          if (status === 'OK') {
            resolve(results?.[0]);
          } else {
            reject(status);
          }
        });
      });

      if (!result) return;

      const addressComponents = (result as google.maps.GeocoderResult).address_components;
      let streetNumber = '',
        route = '',
        city = '',
        state = '',
        country = '',
        postalCode = '';

      for (const component of addressComponents) {
        const types = component.types;
        if (types.includes('street_number')) {
          streetNumber = component.long_name;
        } else if (types.includes('route')) {
          route = component.long_name;
        } else if (
          types.includes('locality') ||
          types.includes('sublocality') ||
          types.includes('neighborhood')
        ) {
          if (!city) {
            city = component.long_name;
          }
        } else if (types.includes('administrative_area_level_1')) {
          state = component.short_name;
        } else if (types.includes('country')) {
          country = component.short_name;
        } else if (types.includes('postal_code')) {
          postalCode = component.long_name;
        }
      }

      if (!city) {
        const cityComponent = addressComponents.find((component) =>
          component.types.some((type) =>
            ['sublocality_level_1', 'neighborhood', 'postal_town'].includes(type)
          )
        );
        if (cityComponent) {
          city = cityComponent.long_name;
        }
      }

      // Update form fields
      form.setValue('address_line1', `${streetNumber} ${route}`.trim());
      form.setValue('city', city);
      form.setValue('state', state);
      form.setValue('country', country);
      form.setValue('postal_code', postalCode);
    } catch (error) {
      logger.error('Error fetching address details:', error);
      toast({
        title: 'Error',
        description: 'Failed to fetch address details. Please try entering manually.',
        variant: 'destructive',
      });
    }
  };

  const onSubmit = async (data: OnboardingFormValues) => {
    if (!data) {
      toast({
        title: 'Error',
        description: 'No form data available. Please try again.',
        variant: 'destructive',
      });
      return;
    }

    setIsSubmitting(true);

    try {
      const requestBody = {
        ...data,
        finished_onboarding: true,
      };

      // Authentication headers will be automatically injected by middleware
      const headers = {
        'Content-Type': 'application/json',
      };

      const response = await fetch('/api/accounts/onboard', {
        method: 'POST',
        headers,
        body: JSON.stringify(requestBody),
      });

      if (!response.ok) {
        let errorMessage = 'Failed to update account';
        try {
          await response.text();
          errorMessage = `Failed to update account: ${response.status} ${response.statusText}`;
        } catch {
          // Error parsing error response
        }
        throw new Error(errorMessage);
      }

      await response.json();

      // Note: Authentication cookie is httpOnly and will be updated by the backend API response

      toast({
        title: 'Success',
        description: 'Your profile has been updated successfully.',
      });

      // Check for redirect parameter or go to dashboard
      const urlParams = new URLSearchParams(window.location.search);
      const redirectTo = urlParams.get('redirect') || '/merchants/dashboard';
      router.push(redirectTo);
    } catch (error) {
      logger.error('Form submission error:', {
        name: error instanceof Error ? error.name : 'Unknown',
        message: error instanceof Error ? error.message : 'Unknown error',
        stack: error instanceof Error ? error.stack : undefined,
      });

      toast({
        title: 'Error',
        description:
          error instanceof Error
            ? error.message
            : 'Failed to update your profile. Please try again.',
        variant: 'destructive',
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <>
      {googleMapsApiKey && (
        <Script
          src={`https://maps.googleapis.com/maps/api/js?key=${googleMapsApiKey}&libraries=places`}
          onLoad={() => setIsGoogleLoaded(true)}
          onError={(e) => {
            logger.error('Error loading Google Maps:', e);
            setScriptError('Failed to load Google Maps');
            toast({
              title: 'Error',
              description:
                'Failed to load Google Maps. Address autocomplete will not be available.',
              variant: 'destructive',
            });
          }}
          strategy="lazyOnload"
        />
      )}
      <Card className="w-full max-w-2xl mx-auto">
        <CardHeader>
          <CardTitle>Complete Your Profile</CardTitle>
          <CardDescription>
            Please provide your details to complete the onboarding process.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Form {...form}>
            <form
              onSubmit={(e) => {
                form.handleSubmit(onSubmit)(e);
              }}
              className="space-y-6"
            >
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <FormField
                  control={form.control}
                  name="first_name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>First Name</FormLabel>
                      <FormControl>
                        <Input placeholder="John" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="last_name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Last Name</FormLabel>
                      <FormControl>
                        <Input placeholder="Doe" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>

              {!scriptError && (
                <FormItem>
                  <FormLabel>Search Address</FormLabel>
                  <FormControl>
                    {isGoogleLoaded && googleMapsApiKey && (
                      <DynamicGooglePlacesAutocomplete
                        apiKey={googleMapsApiKey}
                        selectProps={{
                          className: 'w-full',
                          placeholder: 'Start typing your address...',
                          onChange: handlePlaceSelect,
                        }}
                      />
                    )}
                  </FormControl>
                  <FormDescription>
                    Search for your address or enter details manually below
                  </FormDescription>
                </FormItem>
              )}

              <FormField
                control={form.control}
                name="address_line1"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Address Line 1</FormLabel>
                    <FormControl>
                      <Input placeholder="123 Main St" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="address_line2"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Address Line 2</FormLabel>
                    <FormControl>
                      <Input placeholder="Apt 4B" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <FormField
                  control={form.control}
                  name="city"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>City</FormLabel>
                      <FormControl>
                        <Input placeholder="New York" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="state"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>State/Province/Region</FormLabel>
                      <FormControl>
                        <Input placeholder="NY" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="postal_code"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Postal/ZIP Code</FormLabel>
                      <FormControl>
                        <Input placeholder="10001" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>

              <FormField
                control={form.control}
                name="country"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Country Code</FormLabel>
                    <FormControl>
                      <Input placeholder="US" maxLength={2} {...field} />
                    </FormControl>
                    <FormDescription>Two-letter country code (e.g., US, GB, FR)</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="wallet_address"
                render={({ field }) => {
                  const value = field.value || '';
                  const hasValue = value !== '';
                  const isValidLength = value.length === 42;
                  const showLengthError = hasValue && !isValidLength;
                  const showLengthSuccess = hasValue && isValidLength;

                  return (
                    <FormItem>
                      <FormLabel>Wallet Address (Optional)</FormLabel>
                      <FormControl>
                        <Input
                          placeholder="0x..."
                          className={`font-mono ${showLengthError ? 'border-red-500' : ''} ${showLengthSuccess ? 'border-green-500' : ''}`}
                          {...field}
                          onChange={(e) => {
                            const value = e.target.value;
                            // Remove spaces and ensure lowercase
                            const cleanValue = value.trim().toLowerCase();
                            field.onChange(cleanValue);
                          }}
                        />
                      </FormControl>
                      <FormDescription className="flex flex-col gap-1">
                        <span>Your Wallet address for receiving payments</span>
                        <span className="text-xs text-muted-foreground">
                          Must start with &ldquo;0x&rdquo; followed by 40 hexadecimal characters
                        </span>
                        {showLengthError}
                        {showLengthSuccess}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  );
                }}
              />

              <div className="flex justify-center pt-4">
                <Button type="submit" disabled={isSubmitting} className="w-full md:w-auto">
                  {isSubmitting ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Saving...
                    </>
                  ) : (
                    'Save Details'
                  )}
                </Button>
              </div>
            </form>
          </Form>
        </CardContent>
      </Card>
    </>
  );
}
