'use client';

import { useCallback, useEffect, useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Icons } from '@/components/icons';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle
} from '@/components/ui/sheet';
import { useAppForm } from '@/components/ui/tanstack-form';
import { CastorApiError } from '@/lib/api-client';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { bindAccountContactMutation, sendAccountContactCodeMutation } from '../api/mutations';
import type { ContactType } from '../api/types';
import {
  getBindContactSchema,
  MOBILE_PATTERN,
  type BindContactFormValues
} from '../schemas/account';

const RESEND_COOLDOWN_SECONDS = 60;
const HTTP_CONFLICT = 409;

interface ContactFormSheetProps {
  contactType: ContactType;
  currentContact: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ContactFormSheet({
  contactType,
  currentContact,
  open,
  onOpenChange
}: ContactFormSheetProps) {
  const t = useTranslations('profile.contact');
  const tc = useTranslations('common');
  const isEmail = contactType === 'EMAIL';
  const [countdown, setCountdown] = useState(0);
  const [errorMessage, setErrorMessage] = useState('');

  const schemaMessages = {
    contactRequired: t('validation.contactRequired'),
    invalidEmail: t('validation.invalidEmail'),
    invalidMobile: t('validation.invalidMobile'),
    codeRequired: t('validation.codeRequired')
  };

  const sendCodeMutation = useMutation({
    ...mergeMutationOptions(sendAccountContactCodeMutation, {
      onSuccess: () => {
        setErrorMessage('');
        setCountdown(RESEND_COOLDOWN_SECONDS);
        toast.success(t('messages.codeSent'));
      },
      onError: (error) => {
        const message = resolveErrorMessage(
          error,
          t('messages.alreadyBound'),
          t('messages.codeSendFailed')
        );
        setErrorMessage(message);
        toast.error(message);
      }
    })
  });

  const bindMutation = useMutation({
    ...mergeMutationOptions(bindAccountContactMutation, {
      onSuccess: () => {
        setErrorMessage('');
        toast.success(t('messages.bound'));
        onOpenChange(false);
      },
      onError: (error) => {
        const message = resolveErrorMessage(
          error,
          t('messages.alreadyBound'),
          t('messages.bindFailed')
        );
        setErrorMessage(message);
        toast.error(message);
      }
    })
  });

  const form = useAppForm({
    defaultValues: { contact: currentContact, code: '' } as BindContactFormValues,
    validators: { onSubmit: getBindContactSchema(contactType, schemaMessages) },
    onSubmit: async ({ value }) => {
      await bindMutation.mutateAsync({
        contactType,
        contact: value.contact.trim(),
        code: value.code.trim()
      });
    }
  });

  // Reset the form and the cooldown whenever the sheet is reopened
  useEffect(() => {
    if (!open) return;
    setErrorMessage('');
    setCountdown(0);
    form.reset({ contact: currentContact, code: '' });
  }, [open, currentContact, form]);

  useEffect(() => {
    if (countdown <= 0) return;
    const timer = setInterval(() => setCountdown((prev) => Math.max(0, prev - 1)), 1000);
    return () => clearInterval(timer);
  }, [countdown]);

  const handleSendCode = useCallback(() => {
    const contact = form.state.values.contact.trim();
    if (!contact) {
      setErrorMessage(schemaMessages.contactRequired);
      return;
    }
    if (isEmail ? !contact.includes('@') : !MOBILE_PATTERN.test(contact)) {
      setErrorMessage(isEmail ? schemaMessages.invalidEmail : schemaMessages.invalidMobile);
      return;
    }
    sendCodeMutation.mutate({ contactType, contact });
  }, [
    contactType,
    form,
    isEmail,
    schemaMessages.contactRequired,
    schemaMessages.invalidEmail,
    schemaMessages.invalidMobile,
    sendCodeMutation
  ]);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className='flex flex-col'>
        <SheetHeader>
          <SheetTitle>{isEmail ? t('bindEmailTitle') : t('bindMobileTitle')}</SheetTitle>
          <SheetDescription>
            {isEmail ? t('bindEmailDescription') : t('bindMobileDescription')}
          </SheetDescription>
        </SheetHeader>

        <div className='flex-1 overflow-auto px-1'>
          <form.AppForm>
            <form.Form id='account-contact-form' className='gap-4 p-0 md:p-0'>
              <div className='flex items-end gap-2'>
                <div className='flex-1'>
                  <form.TextField
                    name='contact'
                    label={isEmail ? t('email') : t('mobile')}
                    required
                    type={isEmail ? 'email' : 'tel'}
                    autoComplete={isEmail ? 'email' : 'tel'}
                    placeholder={isEmail ? t('emailPlaceholder') : t('mobilePlaceholder')}
                  />
                </div>
                <Button
                  type='button'
                  variant='outline'
                  className='mb-1'
                  disabled={countdown > 0 || sendCodeMutation.isPending}
                  isLoading={sendCodeMutation.isPending}
                  onClick={handleSendCode}
                >
                  {countdown > 0 ? `${countdown}s` : t('sendCode')}
                </Button>
              </div>
              <form.TextField
                name='code'
                label={t('code')}
                required
                maxLength={10}
                autoComplete='one-time-code'
                placeholder={t('codePlaceholder')}
              />
              {errorMessage && (
                <p className='text-destructive text-sm' role='alert'>
                  {errorMessage}
                </p>
              )}
            </form.Form>
          </form.AppForm>
        </div>

        <SheetFooter>
          <Button type='button' variant='outline' onClick={() => onOpenChange(false)}>
            {tc('cancel')}
          </Button>
          <Button type='submit' form='account-contact-form' isLoading={bindMutation.isPending}>
            <Icons.check /> {t('submit')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

function resolveErrorMessage(error: unknown, conflictMessage: string, fallback: string): string {
  if (error instanceof CastorApiError) {
    if (error.httpStatus === HTTP_CONFLICT) return conflictMessage;
    return error.message || fallback;
  }
  return fallback;
}
