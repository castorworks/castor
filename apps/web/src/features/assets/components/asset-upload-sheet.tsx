'use client';

import { useCallback, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
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
import { createAssetMutation } from '../api/mutations';
import { toast } from 'sonner';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { AssetDropzone } from './asset-dropzone';

interface AssetUploadSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const emptyMetadata = { name: '', description: '', tags: '', folderPath: '', isPublic: false };

export function AssetUploadSheet({ open, onOpenChange }: AssetUploadSheetProps) {
  const t = useTranslations('assets');
  const tc = useTranslations('common');
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [metadata, setMetadata] = useState(emptyMetadata);

  const reset = useCallback(() => {
    setSelectedFile(null);
    setMetadata(emptyMetadata);
  }, []);

  const uploadMutation = useMutation({
    ...mergeMutationOptions(createAssetMutation, {
      onSuccess: (asset) => {
        // 命中去重时后端返回已有资产，本次填写的名称与可见性不会生效，必须明确告知。
        if (asset.isDuplicate) {
          toast.info(t('messages.uploadDuplicate', { name: asset.name }));
        } else {
          toast.success(t('messages.uploadSuccess'));
        }
        reset();
        onOpenChange(false);
      },
      onError: (error) => {
        toast.error(t('messages.uploadFailed', { message: error.message }));
      }
    })
  });

  const handleFileChange = useCallback((file: File | null) => {
    setSelectedFile(file);
    if (file) {
      // Auto-fill name from filename if user hasn't set it
      setMetadata((prev) => ({
        ...prev,
        name: prev.name.trim() || file.name.replace(/\.[^/.]+$/, '')
      }));
    }
  }, []);

  const handleUpload = useCallback(() => {
    if (!selectedFile) return;

    const formData = new FormData();
    formData.append('file', selectedFile);
    if (metadata.name) formData.append('name', metadata.name);
    if (metadata.description) formData.append('description', metadata.description);
    if (metadata.tags) formData.append('tags', metadata.tags);
    if (metadata.folderPath) formData.append('folderPath', metadata.folderPath);
    if (metadata.isPublic) formData.append('isPublic', 'true');

    uploadMutation.mutate(formData);
  }, [selectedFile, metadata, uploadMutation]);

  const handleOpenChange = useCallback(
    (next: boolean) => {
      if (!next) reset();
      onOpenChange(next);
    },
    [onOpenChange, reset]
  );

  const setField = (key: 'name' | 'description' | 'tags' | 'folderPath') => (value: string) =>
    setMetadata((prev) => ({ ...prev, [key]: value }));

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetContent className='flex flex-col'>
        <SheetHeader>
          <SheetTitle>{t('upload.title')}</SheetTitle>
          <SheetDescription>{t('upload.description')}</SheetDescription>
        </SheetHeader>

        <div className='flex-1 overflow-auto px-1'>
          <AssetDropzone
            className='mb-4'
            file={selectedFile}
            onFileChange={handleFileChange}
            disabled={uploadMutation.isPending}
          />

          {/* Metadata fields */}
          {selectedFile && (
            <div className='space-y-4'>
              <div className='space-y-2'>
                <Label htmlFor='asset-name'>{tc('name')}</Label>
                <Input
                  id='asset-name'
                  placeholder={t('form.assetName')}
                  value={metadata.name}
                  onChange={(e) => setField('name')(e.target.value)}
                />
              </div>
              <div className='space-y-2'>
                <Label htmlFor='asset-description'>{tc('description')}</Label>
                <Input
                  id='asset-description'
                  placeholder={t('form.describe')}
                  value={metadata.description}
                  onChange={(e) => setField('description')(e.target.value)}
                />
              </div>
              <div className='space-y-2'>
                <Label htmlFor='asset-tags'>{tc('tags')}</Label>
                <Input
                  id='asset-tags'
                  placeholder={t('form.tags')}
                  value={metadata.tags}
                  onChange={(e) => setField('tags')(e.target.value)}
                />
              </div>
              <div className='space-y-2'>
                <Label htmlFor='asset-folder'>{t('form.folderPath')}</Label>
                <Input
                  id='asset-folder'
                  placeholder='/path/to/folder'
                  value={metadata.folderPath}
                  onChange={(e) => setField('folderPath')(e.target.value)}
                />
              </div>
              <div className='flex items-center justify-between space-x-2'>
                <div>
                  <Label htmlFor='asset-public'>{tc('public')}</Label>
                  <p className='text-muted-foreground text-xs'>{t('form.publicAccess')}</p>
                </div>
                <Switch
                  id='asset-public'
                  checked={metadata.isPublic}
                  onCheckedChange={(isPublic) => setMetadata((prev) => ({ ...prev, isPublic }))}
                />
              </div>
            </div>
          )}
        </div>

        <SheetFooter>
          <Button type='button' variant='outline' onClick={() => handleOpenChange(false)}>
            {tc('cancel')}
          </Button>
          <Button
            onClick={handleUpload}
            disabled={!selectedFile}
            isLoading={uploadMutation.isPending}
          >
            <Icons.upload className='mr-2 h-4 w-4' /> {tc('upload')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

// ============================================================
// Trigger Button
// ============================================================

export function AssetUploadSheetTrigger() {
  const t = useTranslations('assets');
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button onClick={() => setOpen(true)}>
        <Icons.upload className='mr-2 h-4 w-4' /> {t('upload.title')}
      </Button>
      <AssetUploadSheet open={open} onOpenChange={setOpen} />
    </>
  );
}
