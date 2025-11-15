# Supabase Storage RLS Setup Guide

This guide explains how to use **Publishable Key + RLS policies** instead of the Service Role Key for storage uploads.

---

## 🔐 Why Use RLS Instead of Service Role Key?

### **Service Role Key** (Current)
- ✅ Works immediately
- ❌ Bypasses ALL security
- ❌ Has god-mode access to everything
- ⚠️ If leaked, attacker has full access

### **Publishable Key + RLS** (Recommended)
- ✅ Proper security boundaries
- ✅ Fine-grained access control
- ✅ Production-ready
- ✅ If leaked, limited by RLS policies

---

## 📋 Setup Steps

### **Step 1: Create Storage Bucket** (if not exists)

1. Go to **Supabase Dashboard** → **Storage**
2. Click **New Bucket**
3. Name: `hdr-images`
4. Public: `true` (so images are publicly accessible)
5. Click **Create**

---

### **Step 2: Set Up RLS Policies**

Go to **Storage** → **Policies** → Select `hdr-images` bucket

#### **Policy 1: Allow Authenticated Users to Upload**

```sql
-- Policy Name: Allow authenticated uploads
-- Allowed operation: INSERT
-- Target roles: authenticated

CREATE POLICY "Allow authenticated uploads"
ON storage.objects
FOR INSERT
TO authenticated
WITH CHECK (
  bucket_id = 'hdr-images' AND
  auth.uid()::text = (storage.foldername(name))[1]
);
```

**What this does:**
- Allows any authenticated user to upload
- Only to their own folder: `users/{their-user-id}/...`
- Extracts user ID from path and verifies it matches the authenticated user

#### **Policy 2: Allow Public Reads**

```sql
-- Policy Name: Allow public reads
-- Allowed operation: SELECT
-- Target roles: public, authenticated

CREATE POLICY "Allow public reads"
ON storage.objects
FOR SELECT
TO public
USING (bucket_id = 'hdr-images');
```

**What this does:**
- Anyone can view/download images (no auth required)
- Good for sharing images with clients

#### **Policy 3: Allow Users to Delete Their Own Files**

```sql
-- Policy Name: Allow users to delete own files
-- Allowed operation: DELETE
-- Target roles: authenticated

CREATE POLICY "Allow users to delete own files"
ON storage.objects
FOR DELETE
TO authenticated
USING (
  bucket_id = 'hdr-images' AND
  auth.uid()::text = (storage.foldername(name))[1]
);
```

**What this does:**
- Users can only delete files in their own folder
- Prevents users from deleting other users' files

---

### **Step 3: Update Backend Code**

#### **Option A: Keep Service Role Key** (Current)
No changes needed. Uses `SUPABASE_SERVICE_ROLE_KEY`.

#### **Option B: Switch to Publishable Key + RLS**

**Update `cmd/server/main.go`:**

```go
// Use Publishable Key with RLS policies (more secure)
storageClient, err := supabase.NewStorageClient(
    cfg.SupabaseURL, 
    cfg.SupabasePublishableKey,  // ← Changed from ServiceRoleKey
    cfg.SupabaseStorageBucket,
)
```

**Update `internal/supabase/storage.go`:**

The current implementation needs the user's JWT token for RLS to work. You'll need to pass the user's token to storage operations.

**Before:**
```go
func NewStorageClient(supabaseURL, serviceRoleKey, bucket string) (*StorageClient, error) {
    client := storage.NewClient(baseURL+"/storage/v1", serviceRoleKey, nil)
    // ...
}
```

**After (for RLS):**
```go
func NewStorageClient(supabaseURL, apiKey, bucket string) (*StorageClient, error) {
    client := storage.NewClient(baseURL+"/storage/v1", apiKey, nil)
    // ...
}

// For RLS to work, need to pass user's JWT token in each request
func (s *StorageClient) UploadFileWithAuth(userID, orderID uuid.UUID, filename string, data []byte, userToken string) (string, string, error) {
    storagePath := fmt.Sprintf("users/%s/orders/%s/%s", userID.String(), orderID.String(), filename)
    
    contentType := "image/jpeg"
    upsert := true
    
    // Create custom headers with user's JWT token
    headers := map[string]string{
        "Authorization": "Bearer " + userToken,
    }
    
    _, err := s.client.UploadFile(s.bucket, storagePath, bytes.NewReader(data), storage.FileOptions{
        ContentType: &contentType,
        Upsert:      &upsert,
        Headers:     headers,  // Pass user's token for RLS
    })
    
    if err != nil {
        return "", "", fmt.Errorf("failed to upload file: %w", err)
    }
    
    publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s",
        s.baseURL, s.bucket, storagePath)
    
    return storagePath, publicURL, nil
}
```

**Update handlers to pass user token:**

```go
// In internal/handlers/images.go
func (h *ImagesHandler) DownloadImage(c *gin.Context) {
    // ... existing code ...
    
    // Get user's JWT token from request
    userToken := c.GetHeader("Authorization")
    userToken = strings.TrimPrefix(userToken, "Bearer ")
    
    // Upload with user's token for RLS
    _, publicURL, err := h.storageClient.UploadFileWithAuth(
        userID, 
        orderID, 
        filename, 
        imageData,
        userToken,  // ← Pass user's token
    )
    // ...
}
```

---

## ⚖️ Comparison

### **File Paths Structure**
Both approaches use the same structure:
```
hdr-images/
  users/
    {user-id}/
      orders/
        {order-id}/
          img_abc123_high.jpg
```

### **Security Comparison**

| Aspect | Service Role Key | Publishable + RLS |
|--------|------------------|-------------------|
| Setup Complexity | ✅ Easy | ⚠️ Medium (requires RLS) |
| Security | ❌ Low (bypasses all) | ✅ High (policy-based) |
| User Isolation | ⚠️ Must enforce in code | ✅ Enforced by database |
| Audit Trail | ❌ Shows as "service_role" | ✅ Shows actual user |
| Multi-tenancy | ⚠️ Trust backend code | ✅ Database-enforced |

---

## 🚀 Recommendation

### **For Development/MVP:**
✅ Use **Service Role Key** (current setup)
- Faster to get started
- Fewer moving parts
- Good enough for early testing

### **For Production:**
✅ Switch to **Publishable Key + RLS**
- Better security posture
- Proper user isolation
- Scales better with multiple users
- Industry best practice

---

## 🔍 Verify RLS Policies Work

After setting up RLS, test with:

```bash
# This should FAIL (no auth token)
curl -X POST \
  'https://your-project.supabase.co/storage/v1/object/hdr-images/test.jpg' \
  -H 'Authorization: Bearer YOUR_PUBLISHABLE_KEY' \
  --data-binary '@test.jpg'

# This should SUCCEED (with user JWT)
curl -X POST \
  'https://your-project.supabase.co/storage/v1/object/hdr-images/users/{user-id}/test.jpg' \
  -H 'Authorization: Bearer USER_JWT_TOKEN' \
  --data-binary '@test.jpg'
```

---

## 📚 Additional Resources

- [Supabase Storage RLS Docs](https://supabase.com/docs/guides/storage/security/access-control)
- [Row Level Security Guide](https://supabase.com/docs/guides/auth/row-level-security)
- [Storage Policies](https://supabase.com/docs/guides/storage/security/policies)

---

## ✨ Summary

**Current Setup (Service Role Key):**
- Works out of the box ✅
- Less secure ⚠️
- Good for development 👍

**RLS Setup (Publishable Key):**
- Requires policy configuration 🔧
- Much more secure 🔒
- Production-ready 🚀

**Your choice depends on:**
- Development stage (MVP vs Production)
- Security requirements
- Time available for setup

Both are valid! Start with service role for speed, migrate to RLS when ready for production. 🎯

