'use client';

import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { useAppForm } from '@/components/ui/tanstack-form';
import { Button } from '@/components/ui/button';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle
} from '@/components/ui/sheet';
import { Icons } from '@/components/icons';
import { locales } from '@/i18n/config';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { saveOIDCProviderMutation } from '../api/mutations';
import type { OIDCProvider } from '../api/types';
import { providerSchema, type ProviderFormValues } from '../schemas/provider';

interface ProviderFormSheetProps {
  provider?: OIDCProvider;
  onClose: () => void;
}

export function ProviderFormSheet({ provider, onClose }: ProviderFormSheetProps) {
  const t = useTranslations('sso');
  const tc = useTranslations('common');
  const mutation = useMutation(
    mergeMutationOptions(saveOIDCProviderMutation, {
      onSuccess: () => {
        toast.success(t('messages.saved'));
        onClose();
      },
      onError: (error) => toast.error(error.message || t('messages.saveFailed'))
    })
  );

  const form = useAppForm({
    defaultValues: {
      code: provider?.code ?? '',
      ...Object.fromEntries(locales.map((locale) => [locale, provider?.name[locale] ?? ''])),
      issuer: provider?.issuer ?? '',
      clientId: provider?.clientId ?? '',
      clientSecret: '',
      scopes: provider?.scopes.join(' ') ?? 'openid profile email',
      usernameClaim: provider?.usernameClaim ?? 'preferred_username',
      autoRegister: provider?.autoRegister ?? false,
      isEnabled: provider?.isEnabled ?? true,
      sortOrder: provider?.sortOrder ?? 0
    } as ProviderFormValues,
    validators: {
      onSubmit: providerSchema(
        {
          codeInvalid: t('validation.codeInvalid'),
          nameRequired: t('validation.nameRequired'),
          issuerInvalid: t('validation.issuerInvalid'),
          clientIdRequired: t('validation.clientIdRequired'),
          clientSecretRequired: t('validation.clientSecretRequired'),
          scopesInvalid: t('validation.scopesInvalid')
        },
        !provider?.hasClientSecret
      )
    },
    onSubmit: async ({ value }) => {
      await mutation.mutateAsync({
        id: provider?.id,
        data: {
          code: value.code.trim(),
          name: Object.fromEntries(
            locales.map((locale) => [locale, String(value[locale]).trim()])
          ) as OIDCProvider['name'],
          issuer: value.issuer.trim(),
          clientId: value.clientId.trim(),
          clientSecret: value.clientSecret,
          scopes: value.scopes.split(/\s+/).filter(Boolean),
          usernameClaim: value.usernameClaim.trim(),
          autoRegister: value.autoRegister,
          isEnabled: value.isEnabled,
          sortOrder: Number(value.sortOrder) || 0
        }
      });
    }
  });

  return (
    <Sheet open onOpenChange={(open) => !open && onClose()}>
      <SheetContent className='flex w-full flex-col sm:max-w-xl'>
        <SheetHeader>
          <SheetTitle>{provider ? t('form.editTitle') : t('form.createTitle')}</SheetTitle>
          <SheetDescription>{t('form.description')}</SheetDescription>
        </SheetHeader>
        <div className='flex-1 overflow-auto px-1'>
          <form.AppForm>
            <form.Form id='oidc-provider-form' className='space-y-4'>
              <form.TextField
                name='code'
                label={t('form.code')}
                description={t('form.codeDescription')}
                required
                disabled={!!provider}
              />
              <div className='grid gap-3 sm:grid-cols-2'>
                {locales.map((locale) => (
                  <form.TextField
                    key={locale}
                    name={locale}
                    label={t(`form.name_${locale}`)}
                    required
                  />
                ))}
              </div>
              <form.TextField
                name='issuer'
                label={t('form.issuer')}
                description={t('form.issuerDescription')}
                placeholder='https://id.example.com/realms/main'
                required
              />
              <form.TextField name='clientId' label={t('form.clientId')} required />
              <form.TextField
                name='clientSecret'
                label={t('form.clientSecret')}
                type='password'
                description={provider?.hasClientSecret ? t('form.clientSecretKeep') : undefined}
                required={!provider?.hasClientSecret}
              />
              <form.TextField
                name='scopes'
                label={t('form.scopes')}
                description={t('form.scopesDescription')}
              />
              <form.TextField
                name='usernameClaim'
                label={t('form.usernameClaim')}
                description={t('form.usernameClaimDescription')}
              />
              <form.SwitchField
                name='autoRegister'
                label={t('form.autoRegister')}
                description={t('form.autoRegisterDescription')}
              />
              <form.SwitchField
                name='isEnabled'
                label={t('form.enabled')}
                description={t('form.enabledDescription')}
              />
              <form.TextField name='sortOrder' label={t('form.sortOrder')} type='number' />
            </form.Form>
          </form.AppForm>
        </div>
        <SheetFooter>
          <Button variant='outline' onClick={onClose}>
            {tc('cancel')}
          </Button>
          <Button type='submit' form='oidc-provider-form' isLoading={mutation.isPending}>
            <Icons.check /> {tc('save')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
