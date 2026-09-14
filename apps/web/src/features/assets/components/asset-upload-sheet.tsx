'use client';

import { useCallback, useRef, useState } from 'react';
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
import { cn } from '@/lib/utils';
import { mergeMutationOptions } from '@/lib/mutation-utils';

interface AssetUploadSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

export function AssetUploadSheet({ open, onOpenChange }: AssetUploadSheetProps) {
  const t = useTranslations('assets');
  const tc = useTranslations('common');
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [assetName, setAssetName] = useState('');
  const [description, setDescription] = useState('');
  const [tags, setTags] = useState('');
  const [folderPath, setFolderPath] = useState('');
  const [isPublic, setIsPublic] = useState(false);

  const uploadMutation = useMutation({
    ...mergeMutationOptions(createAssetMutation, {
      onSuccess: () => {
        toast.success(t('messages.uploadSuccess'));
        setSelectedFile(null);
        setAssetName('');
        setDescription('');
        setTags('');
        setFolderPath('');
        setIsPublic(false);
        onOpenChange(false);
        if (fileInputRef.current) {
          fileInputRef.current.value = '';
        }
      },
      onError: (error) => {
        toast.error(t('messages.uploadFailed', { message: error.message }));
      }
    })
  });

  const handleFileSelect = useCallback((file: File) => {
    setSelectedFile(file);
    // Auto-fill name from filename if user hasn't set it
    setAssetName((prev) => (prev || '').trim() || file.name.replace(/\.[^/.]+$/, ''));
  }, []);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      setIsDragging(false);
      const files = e.dataTransfer.files;
      if (files.length > 0) {
        handleFileSelect(files[0]);
      }
    },
    [handleFileSelect]
  );

  const handleInputChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const files = e.target.files;
      if (files && files.length > 0) {
        handleFileSelect(files[0]);
      }
    },
    [handleFileSelect]
  );

  const handleUpload = useCallback(() => {
    if (!selectedFile) return;

    const formData = new FormData();
    formData.append('file', selectedFile);
    if (assetName) formData.append('name', assetName);
    if (description) formData.append('description', description);
    if (tags) formData.append('tags', tags);
    if (folderPath) formData.append('folderPath', folderPath);
    if (isPublic) formData.append('isPublic', 'true');

    uploadMutation.mutate(formData);
  }, [selectedFile, assetName, description, tags, folderPath, isPublic, uploadMutation]);

  const handleOpenChange = useCallback(
    (open: boolean) => {
      if (!open) {
        // Reset state when closing
        setSelectedFile(null);
        setAssetName('');
        setDescription('');
        setTags('');
        setFolderPath('');
        setIsPublic(false);
      }
      onOpenChange(open);
    },
    [onOpenChange]
  );

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetContent className='flex flex-col'>
        <SheetHeader>
          <SheetTitle>{t('upload.title')}</SheetTitle>
          <SheetDescription>{t('upload.description')}</SheetDescription>
        </SheetHeader>

        <div className='flex-1 overflow-auto px-1'>
          {/* File dropzone */}
          <div
            className={cn(
              'mb-4 flex flex-col items-center justify-center rounded-lg border-2 border-dashed p-8 transition-colors',
              isDragging
                ? 'border-primary bg-primary/5'
                : 'border-muted-foreground/25 hover:border-muted-foreground/50',
              selectedFile && 'border-primary/50 bg-primary/5'
            )}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
          >
            <input ref={fileInputRef} type='file' className='hidden' onChange={handleInputChange} />
            {selectedFile ? (
              <div className='flex flex-col items-center gap-3 text-center'>
                <Icons.page className='h-12 w-12 text-muted-foreground' />
                <div>
                  <p className='font-medium'>{selectedFile.name}</p>
                  <p className='text-muted-foreground text-sm'>
                    {formatFileSize(selectedFile.size)} &middot;{' '}
                    {selectedFile.type || t('upload.unknownType')}
                  </p>
                </div>
                <Button
                  variant='ghost'
                  size='sm'
                  onClick={() => {
                    setSelectedFile(null);
                    if (fileInputRef.current) {
                      fileInputRef.current.value = '';
                    }
                  }}
                >
                  <Icons.close className='mr-1 h-3 w-3' /> {t('upload.remove')}
                </Button>
              </div>
            ) : (
              <div className='flex flex-col items-center gap-3 text-center'>
                <Icons.upload className='h-10 w-10 text-muted-foreground' />
                <div>
                  <p className='font-medium'>{t('upload.dropHere')}</p>
                  <p className='text-muted-foreground text-sm'>{t('upload.clickToBrowse')}</p>
                </div>
                <Button variant='outline' size='sm' onClick={() => fileInputRef.current?.click()}>
                  {t('upload.browseFiles')}
                </Button>
              </div>
            )}
          </div>

          {/* Metadata fields */}
          {selectedFile && (
            <div className='space-y-4'>
              <div className='space-y-2'>
                <Label htmlFor='asset-name'>{tc('name')}</Label>
                <Input
                  id='asset-name'
                  placeholder={t('form.assetName')}
                  value={assetName}
                  onChange={(e) => setAssetName(e.target.value)}
                />
              </div>
              <div className='space-y-2'>
                <Label htmlFor='asset-description'>{tc('description')}</Label>
                <Input
                  id='asset-description'
                  placeholder={t('form.describe')}
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                />
              </div>
              <div className='space-y-2'>
                <Label htmlFor='asset-tags'>{tc('tags')}</Label>
                <Input
                  id='asset-tags'
                  placeholder={t('form.tags')}
                  value={tags}
                  onChange={(e) => setTags(e.target.value)}
                />
              </div>
              <div className='space-y-2'>
                <Label htmlFor='asset-folder'>{t('form.folderPath')}</Label>
                <Input
                  id='asset-folder'
                  placeholder='/path/to/folder'
                  value={folderPath}
                  onChange={(e) => setFolderPath(e.target.value)}
                />
              </div>
              <div className='flex items-center justify-between space-x-2'>
                <div>
                  <Label htmlFor='asset-public'>{tc('public')}</Label>
                  <p className='text-muted-foreground text-xs'>{t('form.publicAccess')}</p>
                </div>
                <Switch id='asset-public' checked={isPublic} onCheckedChange={setIsPublic} />
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
