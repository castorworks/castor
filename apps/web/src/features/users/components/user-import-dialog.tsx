'use client';

import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Icons } from '@/components/icons';
import { usePasswordPolicy } from '@/hooks/use-password-policy';
import { CastorApiError } from '@/lib/api-client';
import { saveFile, type TableFormat } from '@/lib/download';
import { checkPassword } from '@/lib/password-policy';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { getQueryClient } from '@/lib/query-client';
import { userKeys } from '../api/queries';
import { importUsersMutation } from '../api/mutations';
import { downloadUserImportTemplate } from '../api/service';
import type { UserImportError, UserImportResult } from '../api/types';

const FIELD_KEYS = ['username', 'name', 'email', 'mobile', 'departmentCode'] as const;

/** Header button + dialog: create users in bulk from an Excel or CSV file. */
export function UserImportDialog() {
  const t = useTranslations('users.import');
  const tc = useTranslations('common');
  const { policy, messages, hint } = usePasswordPolicy();
  const [open, setOpen] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [formError, setFormError] = useState<string | null>(null);
  const [rowErrors, setRowErrors] = useState<UserImportError[]>([]);
  const [templatePending, setTemplatePending] = useState(false);

  const reset = () => {
    setFile(null);
    setPassword('');
    setConfirm('');
    setFormError(null);
    setRowErrors([]);
  };

  const mutation = useMutation(
    mergeMutationOptions(importUsersMutation, {
      onSuccess: (result) => {
        toast.success(t('imported', { count: result.created }));
        reset();
        setOpen(false);
      },
      onError: (error) => {
        const data =
          error instanceof CastorApiError ? (error.data as UserImportResult | null) : null;
        if (data?.errors?.length) {
          // A conflict while creating stops midway: the rows before it exist now.
          if (data.created > 0) {
            void getQueryClient().invalidateQueries({ queryKey: userKeys.all });
          }
          setRowErrors(data.errors);
          setFormError(
            data.created > 0 ? t('partiallyImported', { count: data.created }) : error.message
          );
          return;
        }
        setRowErrors([]);
        setFormError(error.message || t('failed'));
      }
    })
  );

  const downloadTemplate = async (format: TableFormat) => {
    setTemplatePending(true);
    try {
      saveFile(await downloadUserImportTemplate(format));
    } catch (error) {
      toast.error(error instanceof Error && error.message ? error.message : t('failed'));
    } finally {
      setTemplatePending(false);
    }
  };

  const submit = () => {
    setRowErrors([]);
    if (!file) {
      setFormError(t('fileRequired'));
      return;
    }
    const issue = checkPassword(password, policy);
    if (issue) {
      setFormError(messages[issue]);
      return;
    }
    if (password !== confirm) {
      setFormError(t('passwordMismatch'));
      return;
    }
    setFormError(null);
    mutation.mutate({ file, password });
  };

  const fieldLabel = (field: string) =>
    (FIELD_KEYS as readonly string[]).includes(field) ? t(`fields.${field}`) : field;

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) reset();
      }}
    >
      <DialogTrigger asChild>
        <Button variant='outline'>
          <Icons.upload /> {t('button')}
        </Button>
      </DialogTrigger>
      <DialogContent className='sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>{t('title')}</DialogTitle>
          <DialogDescription>{t('description')}</DialogDescription>
        </DialogHeader>

        <div className='space-y-5'>
          <div className='flex flex-wrap items-center gap-2'>
            <span className='text-muted-foreground text-sm'>{t('template')}</span>
            <Button
              variant='outline'
              size='sm'
              disabled={templatePending}
              onClick={() => downloadTemplate('xlsx')}
            >
              <Icons.download /> {t('templateExcel')}
            </Button>
            <Button
              variant='outline'
              size='sm'
              disabled={templatePending}
              onClick={() => downloadTemplate('csv')}
            >
              <Icons.download /> {t('templateCsv')}
            </Button>
          </div>

          <div className='space-y-2'>
            <Label htmlFor='user-import-file'>{t('file')}</Label>
            <Input
              id='user-import-file'
              type='file'
              accept='.xlsx,.csv'
              onChange={(event) => {
                setFile(event.target.files?.[0] ?? null);
                setRowErrors([]);
                setFormError(null);
              }}
            />
            <p className='text-muted-foreground text-xs'>{t('fileHint')}</p>
          </div>

          <div className='grid gap-4 sm:grid-cols-2'>
            <div className='space-y-2'>
              <Label htmlFor='user-import-password'>{t('password')}</Label>
              <Input
                id='user-import-password'
                type='password'
                autoComplete='new-password'
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
            </div>
            <div className='space-y-2'>
              <Label htmlFor='user-import-confirm'>{t('confirmPassword')}</Label>
              <Input
                id='user-import-confirm'
                type='password'
                autoComplete='new-password'
                value={confirm}
                onChange={(event) => setConfirm(event.target.value)}
              />
            </div>
            <div className='text-muted-foreground space-y-1 text-xs sm:col-span-2'>
              <p>{hint}</p>
              <p>{t('passwordHint')}</p>
            </div>
          </div>

          {formError && (
            <p role='alert' className='text-destructive text-sm'>
              {formError}
            </p>
          )}

          {rowErrors.length > 0 && (
            <div className='max-h-64 overflow-auto rounded-md border'>
              <table className='w-full text-sm'>
                <thead className='bg-muted/50 sticky top-0'>
                  <tr className='text-left'>
                    <th className='px-3 py-2 font-medium'>{t('errors.line')}</th>
                    <th className='px-3 py-2 font-medium'>{t('errors.field')}</th>
                    <th className='px-3 py-2 font-medium'>{t('errors.problem')}</th>
                  </tr>
                </thead>
                <tbody>
                  {rowErrors.map((error) => (
                    <tr key={`${error.line}-${error.field}`} className='border-t'>
                      <td className='px-3 py-2 tabular-nums'>{error.line}</td>
                      <td className='px-3 py-2'>{fieldLabel(error.field)}</td>
                      <td className='px-3 py-2'>{error.message}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant='outline' onClick={() => setOpen(false)}>
            {tc('cancel')}
          </Button>
          <Button isLoading={mutation.isPending} disabled={mutation.isPending} onClick={submit}>
            <Icons.upload /> {t('submit')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
