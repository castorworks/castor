'use client';

import { useEffect, useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import { Icons } from '@/components/icons';
import { updateSettingMutation, deleteSettingMutation } from '../../api/mutations';
import { useTranslations } from 'next-intl';
import { useSeedTranslation } from '@/lib/use-seed-translation';
import type { LoginMethod, Setting } from '../../api/types';
import { AlertModal } from '@/components/modal/alert-modal';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { usePermission } from '@/hooks/use-permission';

interface SettingCardProps {
  setting: Setting;
}

const TYPE_BADGE_VARIANT: Record<string, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  STRING: 'default',
  NUMBER: 'secondary',
  BOOL: 'outline',
  JSON: 'secondary',
  ARRAY: 'outline',
  SECRET: 'destructive'
};

const LOGIN_METHODS_SETTING_KEY = 'security.login.allowedMethods';
const LOGIN_METHOD_OPTIONS: { value: LoginMethod; labelKey: string }[] = [
  { value: 'password', labelKey: 'settings.loginMethods.password' },
  { value: 'email', labelKey: 'settings.loginMethods.email' },
  { value: 'mobile', labelKey: 'settings.loginMethods.mobile' }
];

function parseLoginMethods(value: string): LoginMethod[] {
  try {
    const parsed = JSON.parse(value) as unknown;
    if (!Array.isArray(parsed)) return ['password', 'email', 'mobile'];
    const selected = LOGIN_METHOD_OPTIONS.map((option) => option.value).filter((method) =>
      parsed.includes(method)
    );
    return selected.length > 0 ? selected : ['password', 'email', 'mobile'];
  } catch {
    return ['password', 'email', 'mobile'];
  }
}

export function SettingCard({ setting }: SettingCardProps) {
  const t = useTranslations('settings');
  const tc = useTranslations('common');
  const { ts } = useSeedTranslation();
  const [editing, setEditing] = useState(false);
  const [localValue, setLocalValue] = useState(setting.value);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const canUpdateSetting = usePermission('/api/v1/admin/settings/:id:PUT');
  const canDeleteSetting = usePermission('/api/v1/admin/settings/:id:DELETE');
  const usesInlineEditor = setting.key === LOGIN_METHODS_SETTING_KEY;
  const canUseGenericEditor = !usesInlineEditor && canUpdateSetting;

  useEffect(() => {
    setLocalValue(setting.value);
  }, [setting.value]);

  const updateMutation = useMutation({
    ...mergeMutationOptions(updateSettingMutation, {
      onSuccess: (_, variables) => {
        const nextValue = variables.values.value;
        if (nextValue !== undefined) {
          setLocalValue(nextValue);
        }
        toast.success(t('messages.updateSingleSuccess'));
        setEditing(false);
      },
      onError: () => toast.error(t('messages.updateSingleFailed'))
    })
  });

  const deleteMutation = useMutation({
    ...mergeMutationOptions(deleteSettingMutation, {
      onSuccess: () => {
        toast.success(t('messages.deleteSuccess'));
        setDeleteOpen(false);
      },
      onError: () => toast.error(t('messages.deleteFailed'))
    })
  });

  const handleSave = () => {
    updateMutation.mutate({
      id: setting.id,
      values: { value: localValue }
    });
  };

  const handleToggle = (checked: boolean) => {
    updateMutation.mutate({
      id: setting.id,
      values: { value: String(checked) }
    });
  };

  const handleLoginMethodsChange = (methods: LoginMethod[]) => {
    updateMutation.mutate({
      id: setting.id,
      values: { value: JSON.stringify(methods) }
    });
  };

  const renderValue = () => {
    if (usesInlineEditor) {
      return (
        <AllowedLoginMethodsEditor
          value={localValue}
          disabled={!canUpdateSetting || updateMutation.isPending}
          onChange={handleLoginMethodsChange}
        />
      );
    }

    if (setting.type === 'BOOL') {
      const boolVal = localValue === 'true';
      return (
        <Switch
          checked={boolVal}
          onCheckedChange={handleToggle}
          disabled={!canUpdateSetting || updateMutation.isPending}
        />
      );
    }

    if (!editing) {
      if (setting.type === 'SECRET') {
        return <span className='text-muted-foreground text-sm'>{'•'.repeat(8)}</span>;
      }
      return (
        <span className='break-all text-sm'>
          {localValue || <span className='text-muted-foreground'>—</span>}
        </span>
      );
    }

    if (setting.type === 'JSON' || setting.type === 'ARRAY') {
      return (
        <Textarea
          value={localValue ?? ''}
          onChange={(e) => setLocalValue(e.target.value)}
          className='font-mono text-sm'
          rows={4}
        />
      );
    }

    if (setting.type === 'NUMBER') {
      return (
        <Input
          type='number'
          value={localValue ?? ''}
          onChange={(e) => setLocalValue(e.target.value)}
        />
      );
    }

    return (
      <Input
        value={localValue ?? ''}
        onChange={(e) => setLocalValue(e.target.value)}
        type={setting.type === 'SECRET' ? 'password' : 'text'}
      />
    );
  };

  return (
    <>
      <AlertModal
        isOpen={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        onConfirm={() => deleteMutation.mutate(setting.id)}
        loading={deleteMutation.isPending}
      />
      <Card>
        <CardHeader className='pb-3'>
          <div className='flex items-start justify-between'>
            <div className='flex flex-col gap-1'>
              <CardTitle className='text-base'>{ts(setting.name)}</CardTitle>
              <CardDescription className='text-xs font-mono'>{setting.key}</CardDescription>
            </div>
            <div className='flex items-center gap-1'>
              <Badge variant={TYPE_BADGE_VARIANT[setting.type] ?? 'secondary'} className='text-xs'>
                {setting.type}
              </Badge>
              {setting.isPublic && (
                <Badge variant='outline' className='text-xs'>
                  {tc('public')}
                </Badge>
              )}
              {setting.isSystem && (
                <Badge variant='secondary' className='text-xs'>
                  {tc('system')}
                </Badge>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {setting.description && (
            <p className='text-muted-foreground mb-3 text-xs'>{ts(setting.description)}</p>
          )}
          <div className='flex items-center gap-2'>
            <div className='flex-1'>{renderValue()}</div>
            {editing && setting.type !== 'BOOL' && canUseGenericEditor && (
              <div className='flex gap-1'>
                <Button
                  size='sm'
                  variant='ghost'
                  onClick={handleSave}
                  disabled={updateMutation.isPending}
                >
                  <Icons.check className='h-4 w-4' />
                </Button>
                <Button
                  size='sm'
                  variant='ghost'
                  onClick={() => {
                    setEditing(false);
                    setLocalValue(setting.value);
                  }}
                >
                  <Icons.close className='h-4 w-4' />
                </Button>
              </div>
            )}
            {!editing && setting.type !== 'BOOL' && canUseGenericEditor && (
              <Button size='sm' variant='ghost' onClick={() => setEditing(true)}>
                <Icons.edit className='h-4 w-4' />
              </Button>
            )}
          </div>
          {!setting.isSystem && canDeleteSetting && (
            <div className='mt-3 flex justify-end'>
              <Button
                size='sm'
                variant='ghost'
                className='text-destructive h-7 px-2'
                onClick={() => setDeleteOpen(true)}
              >
                <Icons.trash className='h-3 w-3' />
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </>
  );
}

function AllowedLoginMethodsEditor({
  value,
  disabled,
  onChange
}: {
  value: string;
  disabled: boolean;
  onChange: (methods: LoginMethod[]) => void;
}) {
  const t = useTranslations();
  const selectedMethods = parseLoginMethods(value);

  return (
    <div className='space-y-2'>
      {LOGIN_METHOD_OPTIONS.map((option) => {
        const checked = selectedMethods.includes(option.value);
        const lastSelected = checked && selectedMethods.length === 1;
        return (
          <div key={option.value} className='flex items-center gap-2'>
            <Checkbox
              id={`login-method-${option.value}`}
              checked={checked}
              disabled={disabled || lastSelected}
              onCheckedChange={(nextChecked) => {
                if (!nextChecked && selectedMethods.length === 1) {
                  toast.error(t('settings.loginMethods.atLeastOne'));
                  return;
                }
                const nextMethods = LOGIN_METHOD_OPTIONS.map((item) => item.value).filter(
                  (method) =>
                    method === option.value
                      ? Boolean(nextChecked)
                      : selectedMethods.includes(method)
                );
                onChange(nextMethods);
              }}
            />
            <Label htmlFor={`login-method-${option.value}`} className='text-sm font-normal'>
              {t(option.labelKey)}
            </Label>
          </div>
        );
      })}
      <p className='text-muted-foreground text-xs'>{t('settings.loginMethods.description')}</p>
    </div>
  );
}
