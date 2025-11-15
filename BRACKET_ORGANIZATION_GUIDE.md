# 📸 Bracket Organization Guide

## Understanding HDR Processing with AutoEnhance AI

### What are Brackets?

**Brackets** are individual photos taken at different exposures (typically underexposed, normal, overexposed). When merged together, they create a High Dynamic Range (HDR) image with better detail in highlights and shadows.

---

## 🎯 Your Current Situation

You have **6 brackets** uploaded with filenames:
- `image_0.jpg` (appears twice)
- `image_1.jpg` (appears twice)  
- `image_2.jpg` (appears twice)

### How to Organize Them?

You have several options for grouping these 6 brackets:

---

## 📋 Bracket Grouping Strategies

### 1. **"auto" Mode (DEFAULT)** ⭐

Groups brackets sequentially by sets of 3 (or your specified number).

**Your 6 brackets → 2 HDR images:**
- Image 1: brackets 1-3 (image_0.jpg #1, image_1.jpg #1, image_2.jpg #1)
- Image 2: brackets 4-6 (image_0.jpg #2, image_1.jpg #2, image_2.jpg #2)

```json
POST /api/v1/orders/{order_id}/process
{
  "enhance_type": "property",
  "bracket_grouping": "auto",
  "brackets_per_image": 3
}
```

**Result:** 2 final HDR images

---

### 2. **"all" Mode**

Merges ALL brackets into ONE HDR image.

**Your 6 brackets → 1 HDR image:**
- Image 1: ALL 6 brackets merged

```json
POST /api/v1/orders/{order_id}/process
{
  "enhance_type": "property",
  "bracket_grouping": "all"
}
```

**Result:** 1 final HDR image (very high dynamic range from 6 exposures)

---

### 3. **"individual" Mode**

Each bracket becomes its own image (no HDR merging).

**Your 6 brackets → 6 individual images:**
- Image 1: image_0.jpg #1
- Image 2: image_1.jpg #1
- Image 3: image_2.jpg #1
- Image 4: image_0.jpg #2
- Image 5: image_1.jpg #2
- Image 6: image_2.jpg #2

```json
POST /api/v1/orders/{order_id}/process
{
  "enhance_type": "property",
  "bracket_grouping": "individual"
}
```

**Result:** 6 final images (enhanced but not HDR merged)

---

### 4. **Custom Grouping**

Specify exactly which brackets to group together using bracket IDs.

**Example: Create 2 HDR images with specific grouping:**

```json
POST /api/v1/orders/{order_id}/process
{
  "enhance_type": "property",
  "bracket_grouping": [
    ["6cb59b4b-407d-4eec-9dba-333a7d54edf8", "5f07a01d-57bc-4b09-abbd-7dcd71c778ef", "c6bebd9d-2eeb-4293-a993-a0233e5d0c2b"],
    ["2dbffdf3-d605-4476-9658-7b89be7b34f2", "f768e971-005f-4b2a-9afd-b02de574930f", "9abbd953-7fb0-4df2-997f-bdd44e5d1d75"]
  ]
}
```

**Result:** 2 final HDR images with your custom grouping

---

## 🔍 How to Get Bracket IDs?

```bash
curl https://your-api.railway.app/api/v1/orders/{order_id}/brackets \
  -H "Authorization: Bearer $JWT_TOKEN"
```

**Response:**
```json
{
  "brackets": [
    {
      "id": "cb8d17c0-4ad9-4139-8348-44d4a72e10a9",
      "bracket_id": "6cb59b4b-407d-4eec-9dba-333a7d54edf8",
      "filename": "image_0.jpg",
      "is_uploaded": true,
      "created_at": "2025-11-15T00:51:58.52775Z"
    }
  ]
}
```

Use the `bracket_id` field for custom grouping.

---

## 💡 Recommended Workflows

### Real Estate Photography (Standard HDR)

```json
{
  "enhance_type": "property",
  "bracket_grouping": "auto",
  "brackets_per_image": 3
}
```

**Why:** Standard 3-exposure HDR provides best results for most real estate shots.

---

### Maximum Dynamic Range

```json
{
  "enhance_type": "property",
  "bracket_grouping": "all"
}
```

**Why:** Use when you have difficult lighting (bright windows + dark interiors) and want maximum detail.

---

### No HDR (Single Exposures)

```json
{
  "enhance_type": "property",
  "bracket_grouping": "individual"
}
```

**Why:** When you don't have bracketed shots or want individual enhancements.

---

## 🎬 Custom Order Names

You can now name your orders when creating them:

```bash
POST /api/v1/orders
{
  "name": "Property Shoot - 123 Main St - Living Room"
}
```

**Response:**
```json
{
  "order_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Property Shoot - 123 Main St - Living Room",
  "status": "pending"
}
```

---

## 📊 Processing Response

After processing, you'll see:

```json
{
  "order_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "processing",
  "message": "Order processing started successfully - Creating 2 HDR image(s) from 6 bracket(s)",
  "processing_params": {
    "enhance_type": "property",
    "total_brackets": 6,
    "total_images": 2,
    "sky_replacement": true,
    "vertical_correction": true,
    "lens_correction": true,
    "window_pull_type": "ONLY_WINDOWS"
  }
}
```

**Key Fields:**
- `total_brackets`: How many brackets were used
- `total_images`: How many final HDR images will be created

---

## 🔄 Realtime Updates

### What Happens During Processing?

1. **You call** `/process` → Order status: `processing`
2. **AutoEnhance processes** → Takes 30 seconds - 5 minutes
3. **Webhook fires** when complete → Backend receives notification
4. **Backend auto-downloads** preview images (watermarked, FREE)
5. **Supabase Realtime publishes** `download_ready` event
6. **iPhone receives** notification with preview URLs

### How to Subscribe (iPhone/Frontend)

```javascript
import { createClient } from '@supabase/supabase-js'

const supabase = createClient('your-supabase-url', 'your-anon-key')

// Subscribe to order updates
const channel = supabase
  .channel(`order:${orderId}`)
  .on('broadcast', { event: 'download_ready' }, (payload) => {
    console.log('Previews ready!', payload)
    // payload contains: { order_id, storage_urls: [...] }
    // Update UI with preview URLs
  })
  .subscribe()
```

### Available Events:

| Event | When | Payload |
|-------|------|---------|
| `upload_started` | Upload begins | `{order_id, timestamp}` |
| `upload_completed` | All brackets uploaded | `{order_id, total_files}` |
| `processing_started` | Processing begins | `{order_id, timestamp}` |
| `processing_progress` | Progress update | `{order_id, progress, total_images}` |
| `processing_completed` | Processing done | `{order_id, total_images}` |
| `processing_failed` | Error occurred | `{order_id, error}` |
| `download_ready` | Previews available | `{order_id, storage_urls: [...]}` |

---

## 🧪 Testing Examples

### Test 1: Auto Mode (Default)
```bash
# 6 brackets → 2 HDR images
curl -X POST https://your-api.railway.app/api/v1/orders/$ORDER_ID/process \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "enhance_type": "property",
    "bracket_grouping": "auto",
    "brackets_per_image": 3
  }'
```

### Test 2: All Mode
```bash
# 6 brackets → 1 HDR image
curl -X POST https://your-api.railway.app/api/v1/orders/$ORDER_ID/process \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "enhance_type": "property",
    "bracket_grouping": "all"
  }'
```

### Test 3: Individual Mode
```bash
# 6 brackets → 6 individual images
curl -X POST https://your-api.railway.app/api/v1/orders/$ORDER_ID/process \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "enhance_type": "property",
    "bracket_grouping": "individual"
  }'
```

---

## ❓ FAQ

### Q: How many brackets should I use per HDR image?
**A:** Standard is 3 (underexposed, normal, overexposed). You can use 5, 7, or more for extreme lighting situations.

### Q: What if my brackets are out of order?
**A:** Use custom grouping with explicit bracket IDs to control the exact grouping.

### Q: Can I mix HDR and non-HDR in one order?
**A:** Yes! Use custom grouping:
```json
{
  "bracket_grouping": [
    ["id1", "id2", "id3"],  // HDR image from 3 brackets
    ["id4"],                 // Single image
    ["id5", "id6"]          // HDR from 2 brackets
  ]
}
```

### Q: What happens if I don't specify bracket_grouping?
**A:** Defaults to `"auto"` with `brackets_per_image: 3`

---

## 🎯 Summary

✅ **Custom order names** - Name your orders when creating  
✅ **Flexible bracket grouping** - Auto, all, individual, or custom  
✅ **Realtime updates** - Webhook → Supabase → iPhone  
✅ **Auto-preview download** - Free watermarked previews automatically  
✅ **Auto-bracket cleanup** - Original brackets deleted after processing  

**Next Steps:**
1. Create an order with a name
2. Upload your brackets
3. Choose your grouping strategy
4. Process and wait for webhook
5. Review previews
6. Download high-res for good ones

Happy Processing! 🚀

