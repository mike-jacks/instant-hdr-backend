# Download Options Reference

## 🎯 Quick Summary

**Watermark defaults to `true` (FREE)** unless explicitly set to `false`.

- ✅ **Watermark = true** → FREE (no credits used)
- 💳 **Watermark = false** → COSTS 1 CREDIT per image

---

## 📋 Available Quality Presets

### 1. **Thumbnail** (400px)

```json
{
  "quality": "thumbnail"
}
```

- Resolution: 400px width
- Use case: List view, tiny previews
- File size: ~50-100KB

### 2. **Preview** (800px) - DEFAULT

```json
{
  "quality": "preview"
}
```

- Resolution: 800px width
- Use case: Gallery view, quick browsing
- File size: ~150-250KB

### 3. **Medium** (1920px)

```json
{
  "quality": "medium"
}
```

- Resolution: 1920px width (Full HD)
- Use case: Full screen mobile, tablet viewing
- File size: ~500KB-1MB

### 4. **High** (Full Resolution)

```json
{
  "quality": "high"
}
```

- Resolution: Original full resolution
- Use case: Client delivery, print, web publishing
- File size: ~2-5MB

### 5. **Custom** (Your Dimensions)

```json
{
  "quality": "custom",
  "max_width": 2400
}
```

**OR**

```json
{
  "quality": "custom",
  "scale": 0.5
}
```

- Resolution: Whatever you specify
- Use case: Specific requirements

---

## 💰 Watermark Options

### FREE Downloads (Default)

```json
{
  "quality": "high"
  // watermark defaults to true = FREE
}
```

**Response:**

```json
{
  "watermark": true,
  "credit_used": false,
  "message": "Image downloaded successfully (FREE with watermark)"
}
```

### Paid Downloads (Unwatermarked)

```json
{
  "quality": "high",
  "watermark": false // COSTS 1 CREDIT
}
```

**Response:**

```json
{
  "watermark": false,
  "credit_used": true,
  "message": "Image downloaded successfully (1 CREDIT USED - unwatermarked)"
}
```

---

## 🎨 Format Options

### JPEG (Default)

```json
{
  "quality": "high",
  "format": "jpeg"
}
```

- Best for photos
- Smaller file sizes
- Most compatible

### PNG

```json
{
  "quality": "high",
  "format": "png"
}
```

- Lossless quality
- Larger file sizes
- Good for graphics

### WebP

```json
{
  "quality": "high",
  "format": "webp"
}
```

- Modern format
- Smaller than JPEG with same quality
- Good browser support

---

## 📱 Complete Examples

### Example 1: FREE Preview for Client Approval

```bash
curl -X POST \
  'https://your-backend.com/api/v1/orders/{order_id}/images/{image_id}/download' \
  -H 'Authorization: Bearer YOUR_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{
    "quality": "high"
  }'
```

**Cost: FREE** (watermark defaults to true)

---

### Example 2: Download Unwatermarked After Approval

```bash
curl -X POST \
  'https://your-backend.com/api/v1/orders/{order_id}/images/{image_id}/download' \
  -H 'Authorization: Bearer YOUR_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{
    "quality": "high",
    "watermark": false
  }'
```

**Cost: 1 CREDIT**

---

### Example 3: Custom Size (2400px) with Watermark

```bash
curl -X POST \
  'https://your-backend.com/api/v1/orders/{order_id}/images/{image_id}/download' \
  -H 'Authorization: Bearer YOUR_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{
    "quality": "custom",
    "max_width": 2400
  }'
```

**Cost: FREE** (watermark defaults to true)

---

### Example 4: PNG Format, Medium Size, No Watermark

```bash
curl -X POST \
  'https://your-backend.com/api/v1/orders/{order_id}/images/{image_id}/download' \
  -H 'Authorization: Bearer YOUR_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{
    "quality": "medium",
    "format": "png",
    "watermark": false
  }'
```

**Cost: 1 CREDIT**

---

### Example 5: 50% Scale with Watermark

```bash
curl -X POST \
  'https://your-backend.com/api/v1/orders/{order_id}/images/{image_id}/download' \
  -H 'Authorization: Bearer YOUR_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{
    "quality": "custom",
    "scale": 0.5
  }'
```

**Cost: FREE** (watermark defaults to true)

---

## 💡 Recommended Workflows

### Workflow A: Cost-Optimized (FREE Preview First)

```
1. Download HIGH quality WITH watermark (FREE)
   → Client reviews full resolution

2. Client approves specific images

3. Download approved images WITHOUT watermark (1 credit each)
```

**Example:**

```json
// Step 1: Free preview
{"quality": "high"}

// Step 2: Paid final (only for approved images)
{"quality": "high", "watermark": false}
```

---

### Workflow B: Multi-Tier Display

```
1. Download THUMBNAIL WITH watermark (FREE)
   → Show in list view

2. User taps image → Download PREVIEW WITH watermark (FREE)
   → Show in detail view

3. User approves → Download HIGH WITHOUT watermark (1 credit)
   → Final delivery
```

**Example:**

```json
// List view
{"quality": "thumbnail"}

// Detail view
{"quality": "preview"}

// Final download
{"quality": "high", "watermark": false}
```

---

## 📊 Cost Comparison

### Scenario: 30 Images Shot

**Approach A: Download All Unwatermarked**

```
Download 30 × high unwatermarked = 30 credits
Client only wants 15 images
Result: Wasted 15 credits
Total Cost: 30 credits
```

**Approach B: FREE Preview First**

```
Download 30 × high watermarked = FREE
Client picks 15 favorites
Download 15 × high unwatermarked = 15 credits
Total Cost: 15 credits (50% savings!)
```

---

## ⚠️ Important Notes

1. **Watermark defaults to TRUE** - You're protected from accidental credit usage
2. **Must explicitly set `watermark: false`** to use credits
3. **Response includes `credit_used` flag** - Always check this
4. **All quality levels support watermark** - thumbnail/preview/medium/high/custom
5. **Format doesn't affect credits** - jpeg/png/webp all same cost

---

## 🎯 Best Practices

✅ **DO:**

- Use watermarked previews for client approval
- Only download unwatermarked for final delivery
- Check `credit_used` in responses
- Use appropriate quality for use case

❌ **DON'T:**

- Download everything unwatermarked upfront
- Set `watermark: false` unless necessary
- Download higher quality than needed
- Forget to check the response message

---

## 🔍 Response Structure

```json
{
  "image_id": "img_abc123",
  "quality": "high",
  "url": "https://supabase.../img_abc123_high.jpg",
  "file_size": 3245678,
  "watermark": true, // ← Was watermark applied?
  "resolution": "full", // ← What resolution?
  "format": "jpeg", // ← What format?
  "credit_used": false, // ← Did this cost a credit?
  "message": "Image downloaded successfully (FREE with watermark) - Quality: high, Resolution: full"
}
```

---

## 🚀 Quick Reference

| Quality     | Resolution  | Typical Size | Use Case      |
| ----------- | ----------- | ------------ | ------------- |
| `thumbnail` | 400px       | 50-100KB     | List view     |
| `preview`   | 800px       | 150-250KB    | Gallery       |
| `medium`    | 1920px      | 500KB-1MB    | Full screen   |
| `high`      | Full        | 2-5MB        | Delivery      |
| `custom`    | Your choice | Varies       | Special needs |

| Watermark        | Cost     | When to Use                     |
| ---------------- | -------- | ------------------------------- |
| `true` (default) | FREE     | Client preview, approval        |
| `false`          | 1 CREDIT | Final delivery, approved images |

---

## ✨ Summary

**Default behavior is SAFE:**

- Watermark = true (FREE)
- Quality = preview (800px)
- Format = jpeg

**To use credits, you must explicitly:**

- Set `"watermark": false`

This protects you from accidentally wasting credits while giving you full flexibility when needed!
