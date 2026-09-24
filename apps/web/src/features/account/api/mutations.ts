import { mutationOptions } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import {
  bindAccountContact,
  changeExpiredPassword,
  resetAccountPassword,
  sendAccountContactCode,
  sendAuthCode,
  updateAccountName,
  updateAccountPassword,
  uploadAccountAvatar,
  updateNotificationPreferences
} from './service';
import { accountKeys } from './queries';
import type {
  BindContactPayload,
  ExpiredPasswordPayload,
  ResetPasswordPayload,
  SendAuthCodePayload,
  SendContactCodePayload,
  UpdateNamePayload,
  UpdatePasswordPayload
} from './types';

export const updateAccountNameMutation = mutationOptions({
  mutationFn: (data: UpdateNamePayload) => updateAccountName(data),
  onSuccess: () => {
    getQueryClient().invalidateQueries({ queryKey: accountKeys.all });
  }
});

export const updateAccountPasswordMutation = mutationOptions({
  mutationFn: (data: UpdatePasswordPayload) => updateAccountPassword(data),
  onSuccess: () => {
    // Password change doesn't invalidate cache, just triggers toast in component
  }
});

export const uploadAccountAvatarMutation = mutationOptions({
  mutationFn: (formData: FormData) => uploadAccountAvatar(formData),
  onSuccess: () => {
    getQueryClient().invalidateQueries({ queryKey: accountKeys.all });
  }
});

export const sendAccountContactCodeMutation = mutationOptions({
  mutationFn: (data: SendContactCodePayload) => sendAccountContactCode(data)
});

export const bindAccountContactMutation = mutationOptions({
  mutationFn: (data: BindContactPayload) => bindAccountContact(data),
  onSuccess: () => {
    getQueryClient().invalidateQueries({ queryKey: accountKeys.all });
  }
});

export const sendAuthCodeMutation = mutationOptions({
  mutationFn: (data: SendAuthCodePayload) => sendAuthCode(data)
});

export const changeExpiredPasswordMutation = mutationOptions({
  mutationFn: (data: ExpiredPasswordPayload) => changeExpiredPassword(data)
});

export const resetAccountPasswordMutation = mutationOptions({
  mutationFn: (data: ResetPasswordPayload) => resetAccountPassword(data)
});

export const updateNotificationPreferencesMutation = mutationOptions({
  mutationFn: (muteNotificationEmails: boolean) =>
    updateNotificationPreferences(muteNotificationEmails),
  onSuccess: () => {
    getQueryClient().invalidateQueries({ queryKey: accountKeys.all });
  }
});
