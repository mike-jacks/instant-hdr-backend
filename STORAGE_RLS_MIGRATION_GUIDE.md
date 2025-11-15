# Storage RLS Migration Guide

## 🎯 What's Been Set Up

The storage RLS policies are now **included as a migration** (`005_storage_rls_policies.sql`), but with special handling because they require elevated privileges.

---

## 🔄 How It Works

### **Scenario A: You Have Elevated Privileges** ✅

When you restart your backend:

```bash
go run cmd/server/main.go
```

**Expected output:**
```
Applying migration: 005_storage_rls_policies.sql
Successfully applied migration: 005_storage_rls_policies.sql
```

✅ **Policies created automatically!** You're done.

---

### **Scenario B: Insufficient Privileges** ⚠️

When you restart your backend:

```bash
go run cmd/server/main.go
```

**Expected output:**
```
Applying migration: 005_storage_rls_policies.sql
⚠️  Warning: Migration 005_storage_rls_policies.sql requires elevated privileges
    Please run this migration manually in Supabase Dashboard SQL Editor:
    File: internal/database/migrations/005_storage_rls_policies.sql
    Continuing with other migrations...
```

✅ **Other migrations still run** - Your backend starts normally.
⚠️ **Action required:** Run the storage policies manually (see below).

---

## 📋 Manual Setup (If Migration Fails)

### **Step 1: Copy the Migration SQL**

Open `internal/database/migrations/005_storage_rls_policies.sql`

### **Step 2: Run in Supabase Dashboard**

1. Go to **https://app.supabase.com/**
2. Select your project
3. Click **SQL Editor**
4. Click **New Query**
5. Paste the contents of `005_storage_rls_policies.sql`
6. Click **Run**

### **Step 3: Mark Migration as Applied**

Run this in SQL Editor to prevent it from trying again:

```sql
INSERT INTO schema_migrations (name, applied_at) 
VALUES ('005_storage_rls_policies.sql', NOW())
ON CONFLICT (name) DO NOTHING;
```

---

## 🔍 Check if Policies Exist

Run this in Supabase SQL Editor:

```sql
SELECT 
    policyname,
    cmd as operation,
    roles
FROM pg_policies 
WHERE tablename = 'objects' 
AND schemaname = 'storage'
AND policyname LIKE '%authenticated%' OR policyname LIKE '%public read%'
ORDER BY policyname;
```

**Should show 4 policies:**
- Allow authenticated users to upload to own folder (INSERT)
- Allow authenticated users to update own files (UPDATE)
- Allow authenticated users to delete own files (DELETE)
- Allow public read access (SELECT)

---

## 🚀 Enable RLS Mode

After policies are created (automatically or manually):

### **Set Environment Variable:**

```bash
# In .env
SUPABASE_USE_RLS=true
```

**OR in Railway:**
1. Go to project → Variables
2. Add: `SUPABASE_USE_RLS` = `true`
3. Deploy

### **Restart Backend:**

```bash
go run cmd/server/main.go

# Should see:
# "Using Supabase Storage with RLS (publishable key) - More secure"
```

---

## ✅ Verify It Works

Try downloading an image:

```bash
POST /api/v1/orders/{order_id}/images/{image_id}/download
{
  "quality": "high"
}
```

**Success indicators:**
- ✅ No "Invalid Compact JWS" error
- ✅ No "must be owner of table objects" error
- ✅ File uploads to: `users/{user-id}/orders/{order-id}/filename.jpg`
- ✅ Returns Supabase Storage URL

---

## 🔄 Migration Behavior Summary

| Situation | What Happens | Action Needed |
|-----------|-------------|---------------|
| ✅ Elevated privileges | Policies created automatically | None - you're done! |
| ⚠️ Insufficient privileges | Warning logged, other migrations continue | Run SQL manually in dashboard |
| ❌ Other migration error | Migration fails, backend won't start | Fix the migration issue |

---

## 🛠️ Troubleshooting

### **Backend won't start**

Check logs for migration errors. The storage migration should NOT prevent startup - it only warns and continues.

### **Policies not working**

1. **Check they exist:**
   ```sql
   SELECT COUNT(*) FROM pg_policies 
   WHERE tablename = 'objects' AND schemaname = 'storage';
   ```
   Should return `4` (or more if you have other policies).

2. **Check RLS is enabled:**
   ```bash
   SUPABASE_USE_RLS=true
   ```

3. **Check bucket exists:**
   - Go to Storage → should see `hdr-images` bucket
   - Bucket should be **Public**

### **"Invalid Compact JWS" error**

- RLS is enabled but policies aren't created yet
- Run the migration SQL manually in dashboard

---

## 🎯 Quick Start Checklist

After updating your code:

1. [ ] Restart backend - migration will try to run
2. [ ] Check logs for migration status
3. [ ] If warning about elevated privileges:
   - [ ] Copy `005_storage_rls_policies.sql` to dashboard
   - [ ] Run it manually
   - [ ] Mark as applied in `schema_migrations`
4. [ ] Set `SUPABASE_USE_RLS=true`
5. [ ] Restart backend
6. [ ] Test image download

---

## 📚 Files Involved

- **Migration:** `internal/database/migrations/005_storage_rls_policies.sql`
- **Migrator:** `internal/database/migrator.go` (handles permission errors gracefully)
- **Config:** `SUPABASE_USE_RLS` environment variable

---

## 🎉 Summary

**Best case:** Migration runs automatically, policies created, RLS works ✅

**Typical case:** Migration warns about privileges, you run SQL manually in dashboard, RLS works ✅

**Either way, your backend won't break!** The migrator is smart enough to continue even if storage policies fail.

