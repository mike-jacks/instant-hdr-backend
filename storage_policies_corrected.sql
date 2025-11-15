-- Storage RLS Policies for hdr-images bucket
-- Run this in Supabase Dashboard → SQL Editor

-- Enable RLS on storage.objects
ALTER TABLE storage.objects ENABLE ROW LEVEL SECURITY;

-- Policy 1: Allow authenticated users to INSERT (upload) to their own folder
CREATE POLICY "Allow authenticated users to upload to own folder"
ON storage.objects FOR INSERT WITH CHECK (
    bucket_id = 'hdr-images'
    AND (storage.foldername(name))[1] = 'users'
    AND (storage.foldername(name))[2] = auth.uid()::text
    AND (select auth.role()) = 'authenticated'
);

-- Policy 2: Allow authenticated users to UPDATE their own files
CREATE POLICY "Allow authenticated users to update own files"
ON storage.objects FOR UPDATE USING (
    bucket_id = 'hdr-images'
    AND (storage.foldername(name))[1] = 'users'
    AND (storage.foldername(name))[2] = auth.uid()::text
    AND (select auth.role()) = 'authenticated'
) WITH CHECK (
    bucket_id = 'hdr-images'
    AND (storage.foldername(name))[1] = 'users'
    AND (storage.foldername(name))[2] = auth.uid()::text
    AND (select auth.role()) = 'authenticated'
);

-- Policy 3: Allow authenticated users to DELETE their own files
CREATE POLICY "Allow authenticated users to delete own files"
ON storage.objects FOR DELETE USING (
    bucket_id = 'hdr-images'
    AND (storage.foldername(name))[1] = 'users'
    AND (storage.foldername(name))[2] = auth.uid()::text
    AND (select auth.role()) = 'authenticated'
);

-- Policy 4: Allow public to SELECT (read/download) all files
CREATE POLICY "Allow public read access"
ON storage.objects FOR SELECT USING (
    bucket_id = 'hdr-images'
);

-- Verify policies were created
SELECT 
    schemaname,
    tablename,
    policyname,
    cmd as operation
FROM pg_policies 
WHERE tablename = 'objects' 
AND schemaname = 'storage'
ORDER BY policyname;

