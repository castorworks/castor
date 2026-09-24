'use client';

import { useQuery } from '@tanstack/react-query';
import { notificationChannelsQueryOptions } from '../api/queries';

import { useMemo, useState } from 'react';
import { typedField, useAppForm } from '@/components/ui/tanstack-form';
import { FormAssetListField } from '@/components/forms/fields';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle
} from '@/components/ui/sheet';
import { Icons } from '@/components/icons';
import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { useDict } from '@/hooks/use-dict';
import { dictOptionsOr } from '@/lib/dict';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { createAdminNotificationMutation, updateAdminNotificationMutation } from '../api/mutations';
import { adminNotificationAttachmentUrl } from '../api/service';
import { usePermission } from '@/hooks/use-permission';
import {
  NOTIFICATION_ATTACHMENT_ADMIN_DOWNLOAD_PERMISSION,
  NOTIFICATION_ATTACHMENTS_MAX,
  NOTIFICATION_ATTACHMENT_UPLOAD_PATH,
  NOTIFICATION_ATTACHMENT_UPLOAD_PERMISSION
} from '../api/types';
import type {
  CreateNotificationPayload,
  Notification,
  NotificationLevel,
  NotificationType,
  UpdateNotificationPayload
} from '../api/types';
import {
  createNotificationSchema,
  type CreateNotificationFormValues
} from '../schemas/notification';

const FormAttachmentsField = typedField<CreateNotificationFormValues>()(FormAssetListField);

interface NotificationFormSheetProps {
  notification?: Notification;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function toDatetimeLocal(value: string | null | undefined): string {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  const offsetMs = date.getTimezoneOffset() * 60_000;
  return new Date(date.getTime() - offsetMs).toISOString().slice(0, 16);
}

function parseRecipientIds(value: string | undefined): number[] {
  if (!value?.trim()) return [];
  return value
    .split(',')
    .map((id) => Number(id.trim()))
    .filter((id) => Number.isInteger(id) && id > 0);
}

function toISOStringOrUndefined(value: string | undefined): string | undefined {
  if (!value) return undefined;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? undefined : date.toISOString();
}

function normalizeOptional(value: string | undefined): string | undefined {
  const trimmed = value?.trim();
  return trimmed ? trimmed : undefined;
}

export function NotificationFormSheet({
  notification,
  open,
  onOpenChange
}: NotificationFormSheetProps) {
  const t = useTranslations('notifications.admin');
  const tc = useTranslations('common');
  const dict = useDict();
  const isEdit = !!notification;

  const schema = useMemo(
    () =>
      createNotificationSchema(t, {
        requireRecipients: !isEdit || notification?.isGlobal
      }),
    [isEdit, notification?.isGlobal, t]
  );

  const createMutation = useMutation({
    ...mergeMutationOptions(createAdminNotificationMutation, {
      onSuccess: () => {
        toast.success(t('messages.createSuccess'));
        onOpenChange(false);
        form.reset();
      },
      onError: (error) => toast.error(error.message || t('messages.createFailed'))
    })
  });

  const updateMutation = useMutation({
    ...mergeMutationOptions(updateAdminNotificationMutation, {
      onSuccess: () => {
        toast.success(t('messages.updateSuccess'));
        onOpenChange(false);
      },
      onError: (error) => toast.error(error.message || t('messages.updateFailed'))
    })
  });

  const form = useAppForm({
    defaultValues: {
      title: notification?.title ?? '',
      content: notification?.content ?? '',
      type: notification?.type ?? 'SYSTEM',
      level: notification?.level ?? 'INFO',
      link: notification?.link ?? '',
      extra: notification?.extra ?? '',
      isGlobal: notification?.isGlobal ?? false,
      sendEmail: notification?.sendEmail ?? false,
      userIds: '',
      expireAt: toDatetimeLocal(notification?.expireAt),
      attachments: notification?.attachments ?? []
    } as CreateNotificationFormValues,
    validators: {
      onSubmit: schema
    },
    onSubmit: async ({ value }) => {
      const v = value as CreateNotificationFormValues;
      const userIds = parseRecipientIds(v.userIds);
      const basePayload = {
        title: v.title.trim(),
        content: v.content ?? '',
        type: v.type as NotificationType,
        level: v.level as NotificationLevel,
        link: normalizeOptional(v.link),
        extra: v.extra ?? '',
        expireAt: toISOStringOrUndefined(v.expireAt),
        // 总是提交完整集合：后端据此同步引用，被移除的附件随之回收。
        attachments: v.attachments.map(({ objectKey, name }) => ({ objectKey, name })),
        sendEmail: v.sendEmail
      };

      if (isEdit && notification) {
        const payload: UpdateNotificationPayload = { ...basePayload };
        if (notification.isGlobal !== v.isGlobal || userIds.length > 0) {
          payload.isGlobal = v.isGlobal;
          payload.userIds = userIds;
        }
        await updateMutation.mutateAsync({ id: notification.id, data: payload });
        return;
      }

      const payload: CreateNotificationPayload = {
        ...basePayload,
        isGlobal: v.isGlobal,
        userIds
      };
      await createMutation.mutateAsync(payload);
    }
  });

  const isPending = createMutation.isPending || updateMutation.isPending;
  const { data: channels } = useQuery(notificationChannelsQueryOptions());
  // An already-on option stays editable (to turn it off) even if SMTP went away.
  const emailAvailable = (channels?.email ?? false) || (notification?.sendEmail ?? false);
  // 只有已经保存在这条通知上的附件能从管理端下载；刚上传的还没挂上，后端会回 404。
  const canDownloadAttachments = usePermission(NOTIFICATION_ATTACHMENT_ADMIN_DOWNLOAD_PERMISSION);
  const saved = new Set(notification?.attachments.map((item) => item.objectKey));
  const attachmentHref = (item: { objectKey: string }) =>
    notification && canDownloadAttachments && saved.has(item.objectKey)
      ? adminNotificationAttachmentUrl(notification.id, item.objectKey)
      : undefined;
  const typeOptions = dictOptionsOr(dict, 'notification_type', [
    { value: 'SYSTEM', label: t('options.types.system') },
    { value: 'ANNOUNCE', label: t('options.types.announce') },
    { value: 'MESSAGE', label: t('options.types.message') },
    { value: 'ALERT', label: t('options.types.alert') },
    { value: 'TASK', label: t('options.types.task') }
  ]);
  const levelOptions = dictOptionsOr(dict, 'notification_level', [
    { value: 'INFO', label: t('options.levels.info') },
    { value: 'SUCCESS', label: t('options.levels.success') },
    { value: 'WARNING', label: t('options.levels.warning') },
    { value: 'ERROR', label: t('options.levels.error') }
  ]);

  return (
    <Sheet key={notification?.id ?? 'create'} open={open} onOpenChange={onOpenChange}>
      <SheetContent className='flex flex-col'>
        <SheetHeader>
          <SheetTitle>{isEdit ? t('form.editTitle') : t('form.newTitle')}</SheetTitle>
          <SheetDescription>
            {isEdit ? t('form.editDescription') : t('form.newDescription')}
          </SheetDescription>
        </SheetHeader>

        <div className='flex-1 overflow-auto px-1'>
          <form.AppForm>
            <form.Form id='notification-form-sheet' className='space-y-4'>
              <form.TextField
                name='title'
                label={t('form.title')}
                required
                placeholder={t('form.titlePlaceholder')}
              />
              <form.TextareaField
                name='content'
                label={t('form.content')}
                placeholder={t('form.contentPlaceholder')}
                className='min-h-28'
              />
              <div className='grid grid-cols-1 gap-4 sm:grid-cols-2'>
                <form.SelectField
                  name='type'
                  label={tc('type')}
                  required
                  options={typeOptions}
                  placeholder={t('form.selectType')}
                />
                <form.SelectField
                  name='level'
                  label={t('form.level')}
                  required
                  options={levelOptions}
                  placeholder={t('form.selectLevel')}
                />
              </div>
              <form.TextField
                name='link'
                label={t('form.link')}
                placeholder={t('form.linkPlaceholder')}
              />
              <form.TextareaField
                name='extra'
                label={t('form.extra')}
                placeholder={t('form.extraPlaceholder')}
                className='min-h-20 font-mono text-xs'
              />
              <form.SwitchField
                name='isGlobal'
                label={t('form.global')}
                description={t('form.globalDescription')}
              />
              <form.TextField
                name='userIds'
                label={t('form.recipients')}
                required={!isEdit || notification?.isGlobal}
                placeholder={t('form.recipientsPlaceholder')}
                description={t(
                  isEdit && !notification?.isGlobal
                    ? 'form.recipientsEditDescription'
                    : 'form.recipientsDescription'
                )}
              />
              {emailAvailable ? (
                <form.SwitchField
                  name='sendEmail'
                  label={t('form.sendEmail')}
                  description={t('form.sendEmailDescription')}
                />
              ) : (
                <p className='text-muted-foreground text-xs'>{t('form.sendEmailUnavailable')}</p>
              )}
              <FormAttachmentsField
                name='attachments'
                label={t('form.attachments')}
                description={t('form.attachmentsDescription')}
                uploadPath={NOTIFICATION_ATTACHMENT_UPLOAD_PATH}
                uploadPermission={NOTIFICATION_ATTACHMENT_UPLOAD_PERMISSION}
                max={NOTIFICATION_ATTACHMENTS_MAX}
                hrefFor={attachmentHref}
              />
              <form.AppField name='expireAt'>
                {(field) => (
                  <field.FieldSet>
                    <field.Field>
                      <field.FieldLabel htmlFor={field.name}>{t('form.expireAt')}</field.FieldLabel>
                      <Input
                        id={field.name}
                        type='datetime-local'
                        value={(field.state.value as string) ?? ''}
                        onBlur={field.handleBlur}
                        onChange={(event) => field.handleChange(event.target.value)}
                      />
                      <field.FieldDescription>
                        {t('form.expireAtDescription')}
                      </field.FieldDescription>
                    </field.Field>
                    <field.FieldError />
                  </field.FieldSet>
                )}
              </form.AppField>
            </form.Form>
          </form.AppForm>
        </div>

        <SheetFooter>
          <Button type='button' variant='outline' onClick={() => onOpenChange(false)}>
            {tc('cancel')}
          </Button>
          <Button type='submit' form='notification-form-sheet' isLoading={isPending}>
            <Icons.check /> {isEdit ? t('form.update') : t('form.create')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

export function NotificationFormSheetTrigger() {
  const t = useTranslations('notifications.admin');
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button onClick={() => setOpen(true)}>
        <Icons.add className='mr-2 h-4 w-4' /> {t('form.create')}
      </Button>
      <NotificationFormSheet open={open} onOpenChange={setOpen} />
    </>
  );
}
