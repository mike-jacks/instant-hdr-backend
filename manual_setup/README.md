# Manual Setup Required

This directory contains setup scripts that require elevated privileges and must be run manually in the Supabase Dashboard.

---

## 📋 Storage RLS Policies

**File:** `storage_rls_policies.sql`

**Why manual?** Storage policies require admin privileges that database migrations don't have.

### **Setup Steps:**

1. Go to **https://app.supabase.com/**
2. Select your project
3. Click **SQL Editor**
4. Click **New Query**
5. Copy the contents of `storage_rls_policies.sql`
6. Click **Run**

### **When to run:**

- ✅ First time setting up the project
- ✅ When `SUPABASE_USE_RLS=true` is enabled
- ✅ Before deploying to production

### **How to verify:**

Run this in SQL Editor:

```sql
SELECT policyname, cmd 
FROM pg_policies 
WHERE tablename = 'objects' 
AND schemaname = 'storage';
```

Should return 4 policies.

---

## 🔐 What These Policies Do

1. **Upload Policy:** Users can only upload to `users/{their-id}/...`
2. **Update Policy:** Users can only update their own files
3. **Delete Policy:** Users can only delete their own files
4. **Read Policy:** Anyone can view/download files (for sharing with clients)

---

## ⚠️ Important Notes

- These policies are **required** for RLS mode (`SUPABASE_USE_RLS=true`)
- Without them, uploads will fail with "Invalid Compact JWS" error
- They only need to be run **once per project**
- Safe to run multiple times (uses `DROP POLICY IF EXISTS`)

---

## 📚 More Information

See the complete guides:
- `RLS_SETUP_COMPLETE_GUIDE.md`
- `SUPABASE_RLS_SETUP.md`

