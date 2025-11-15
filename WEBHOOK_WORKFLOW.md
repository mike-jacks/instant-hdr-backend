# 🎯 Complete AutoEnhance AI Workflow Guide

## Overview

This document describes the complete end-to-end workflow from iPhone upload to final high-res download, including the automatic webhook-driven preview generation and manual cleanup options.

---

## 📱 Workflow Steps

### **1. Create Order**
```bash
POST /api/v1/orders
Authorization: Bearer <jwt_token>

{
  "name": "Client Property - 123 Main St"
}
```

**Response:**
```json
{
  "order_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Client Property - 123 Main St",
  "status": "pending"
}
```

---

### **2. Upload Brackets (Raw Images)**
```bash
POST /api/v1/orders/{order_id}/upload
Authorization: Bearer <jwt_token>
Content-Type: multipart/form-data

files: [IMG_3090.JPG, IMG_3091.JPG, IMG_3092.JPG]
```

**What happens:**
- Backend creates brackets in AutoEnhance AI
- Files are uploaded to AutoEnhance's S3 via pre-signed URLs
- Brackets are verified as uploaded (with retry logic for async processing)

**Response:**
```json
{
  "order_id": "550e8400-e29b-41d4-a716-446655440000",
  "files": [
    {"filename": "IMG_3090.JPG", "size": 2823610},
    {"filename": "IMG_3091.JPG", "size": 3550856},
    {"filename": "IMG_3092.JPG", "size": 3293021}
  ],
  "status": "uploaded"
}
```

---

### **3. Process Order**
```bash
POST /api/v1/orders/{order_id}/process
Authorization: Bearer <jwt_token>

{
  "enhance_type": "property",
  "sky_replacement": true,
  "vertical_correction": true,
  "lens_correction": true,
  "window_pull_type": "ONLY_WINDOWS"
}
```

**What happens:**
- AutoEnhance AI starts processing all brackets
- Order status changes to "processing"
- Real-time event published: `processing_started`

**Response:**
```json
{
  "order_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "processing",
  "message": "Processing started successfully",
  "processing_params": {
    "enhance_type": "property",
    "sky_replacement": true,
    "vertical_correction": true,
    "lens_correction": true,
    "window_pull_type": "ONLY_WINDOWS"
  }
}
```

---

### **4. ⚡ Webhook Auto-Downloads Previews (AUTOMATIC)**

When processing completes, AutoEnhance AI sends a webhook event:

```json
POST /api/v1/webhooks/autoenhance
Authorization: Bearer <webhook_token>

{
  "event": "image_processed",
  "image_id": "img_abc123",
  "error": false,
  "order_id": "550e8400-e29b-41d4-a716-446655440000",
  "order_is_processing": false
}
```

**What happens automatically:**
1. Backend receives webhook notification
2. Backend downloads **PREVIEW images with watermark** (FREE) from AutoEnhance
3. Preview images are uploaded to Supabase Storage
4. Metadata stored in `order_files` table (marked as `is_final=false`)
5. Order status changes to `previews_ready`
6. Real-time event published: `download_ready` with preview URLs
7. **Original brackets are auto-deleted from AutoEnhance** (cleanup)

**No user action needed!** iPhone receives real-time notification with preview URLs.

---

### **5. 🖼️ View Previews on iPhone**
```bash
GET /api/v1/orders/{order_id}/images
Authorization: Bearer <jwt_token>
```

**Response:**
```json
{
  "images": [
    {
      "image_id": "img_abc123",
      "image_name": "enhanced_001",
      "status": "completed",
      "enhance_type": "property",
      "preview_downloaded": true,
      "preview_url": "https://storage.supabase.co/.../preview_img_abc1_20251115.jpg",
      "high_res_downloaded": false,
      "processing_settings": {
        "enhance_type": "property",
        "sky_replacement": true,
        "vertical_correction": true,
        "lens_correction": true,
        "window_pull_type": "ONLY_WINDOWS"
      }
    }
  ]
}
```

**iPhone displays:**
- Low-res watermarked previews
- User can review each image
- Decide which to keep/delete/download high-res

---

### **6a. 🗑️ Delete Bad Preview (Optional)**

If a preview looks bad, delete the image:

```bash
DELETE /api/v1/orders/{order_id}/images/{image_id}
Authorization: Bearer <jwt_token>
```

**What happens:**
- Image deleted from AutoEnhance AI
- All associated files deleted from Supabase Storage
- Database records removed

**Response:**
```json
{
  "message": "Image deleted successfully from AutoEnhance and 2 associated file(s) removed from Supabase",
  "image_id": "img_abc123",
  "deleted_files": 2
}
```

---

### **6b. ⬇️ Download High-Res for Good Previews**

For images you like, download the high-res version:

```bash
POST /api/v1/orders/{order_id}/images/{image_id}/download
Authorization: Bearer <jwt_token>

{
  "quality": "high",
  "watermark": false
}
```

**Quality Options:**
- `thumbnail`: 400px width (~50-100KB) - List view
- `preview`: 800px width (~150-250KB) - Gallery view
- `medium`: 1920px width (~500KB-1MB) - Full screen
- `high`: Full resolution (~2-5MB) - Client delivery ⭐
- `custom`: Specify `max_width` or `scale`

**Watermark Options:**
- `true`: FREE download with watermark
- `false`: **COSTS 1 CREDIT** (unwatermarked)

**What happens:**
- Image downloaded from AutoEnhance AI at specified quality
- Uploaded to Supabase Storage (using user's JWT for RLS)
- Metadata stored in `order_files` table (marked as `is_final=true`)

**Response:**
```json
{
  "image_id": "img_abc123",
  "quality": "high",
  "url": "https://storage.supabase.co/.../img_abc123_high.jpg",
  "file_size": 4532876,
  "watermark": false,
  "resolution": "full",
  "format": "jpeg",
  "credit_used": true,
  "message": "Image downloaded successfully (1 CREDIT USED - unwatermarked) - Quality: high, Resolution: full"
}
```

---

### **7. 📦 Check Order Status Anytime**
```bash
GET /api/v1/orders/{order_id}/status
Authorization: Bearer <jwt_token>
```

**Response:**
```json
{
  "order_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "previews_ready",
  "progress": 100,
  "total_images": 3,
  "total_brackets": 3,
  "uploaded_brackets": 3,
  "autoenhance_status": "waiting",
  "is_processing": false,
  "images": [
    {
      "image_id": "img_abc123",
      "status": "completed",
      "enhance_type": "property"
    }
  ]
}
```

---

## 🔄 Real-time Events (Supabase Realtime)

iPhone subscribes to:
- Channel: `order:{order_id}`
- Channel: `user:{user_id}`

**Events:**
- `upload_started` - Upload begins
- `upload_completed` - All files uploaded
- `processing_started` - AutoEnhance starts processing
- `processing_progress` - Progress updates
- `processing_completed` - Processing finished
- `processing_failed` - Error occurred
- `download_ready` - Preview URLs available ⭐

---

## 🧹 Cleanup & Maintenance

### **Auto-Cleanup (Happens Automatically):**
1. **After successful processing** → Original brackets deleted from AutoEnhance
2. **Webhook handles this** → No user action needed

### **Manual Delete Options:**

#### Delete a Processed Image
```bash
DELETE /api/v1/orders/{order_id}/images/{image_id}
```
- Removes from AutoEnhance AI
- Removes from Supabase Storage
- Removes from database

#### Delete a Bracket (Before Processing)
```bash
DELETE /api/v1/orders/{order_id}/brackets/{bracket_id}
```
- Removes from AutoEnhance AI
- Keeps database record for audit trail

#### Delete Entire Order
```bash
DELETE /api/v1/orders/{order_id}
```
- Removes order from AutoEnhance AI
- Removes all files from Supabase Storage
- Removes all database records

---

## 💰 Cost Summary

| Action | Cost |
|--------|------|
| Upload brackets | FREE |
| Process images | 1 CREDIT per image |
| Preview with watermark | FREE (auto-downloaded) |
| High-res with watermark | FREE (manual download) |
| High-res without watermark | 1 CREDIT (manual download) |
| Delete anything | FREE |

---

## 🎯 Typical User Journey

1. **Create order** → `pending`
2. **Upload 3 photos** → `uploaded`
3. **Hit "Process" button** → `processing`
4. **Wait for webhook** (30 seconds - 5 minutes)
5. **Automatic preview download** → `previews_ready`
6. **iPhone receives notification** with preview URLs
7. **User reviews previews:**
   - Delete bad ones
   - Download high-res for good ones
8. **Final delivery:**
   - High-res unwatermarked images in Supabase
   - Ready for client download

---

## 🧪 Testing the Complete Flow

### Quick Test Script:

```bash
# 1. Create order
ORDER_ID=$(curl -X POST https://your-api.railway.app/api/v1/orders \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Order"}' | jq -r '.order_id')

echo "Order ID: $ORDER_ID"

# 2. Upload files
curl -X POST https://your-api.railway.app/api/v1/orders/$ORDER_ID/upload \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -F "files=@IMG_3090.JPG" \
  -F "files=@IMG_3091.JPG" \
  -F "files=@IMG_3092.JPG"

# 3. Process
curl -X POST https://your-api.railway.app/api/v1/orders/$ORDER_ID/process \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "enhance_type": "property",
    "sky_replacement": true,
    "vertical_correction": true,
    "lens_correction": true,
    "window_pull_type": "ONLY_WINDOWS"
  }'

# 4. Wait for webhook (watch logs)
echo "Waiting for processing to complete..."
sleep 60

# 5. Check images
curl https://your-api.railway.app/api/v1/orders/$ORDER_ID/images \
  -H "Authorization: Bearer $JWT_TOKEN" | jq

# 6. Download high-res for first image
IMAGE_ID=$(curl -s https://your-api.railway.app/api/v1/orders/$ORDER_ID/images \
  -H "Authorization: Bearer $JWT_TOKEN" | jq -r '.images[0].image_id')

curl -X POST https://your-api.railway.app/api/v1/orders/$ORDER_ID/images/$IMAGE_ID/download \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"quality": "high", "watermark": false}' | jq
```

---

## 🚨 Troubleshooting

### Webhook not firing?
- Check `AUTOENHANCE_WEBHOOK_TOKEN` is set correctly
- Verify webhook URL in AutoEnhance dashboard: `https://your-api.railway.app/api/v1/webhooks/autoenhance`
- Check server logs for webhook errors

### Previews not showing?
- Verify order status is `previews_ready`
- Check Supabase Storage bucket has files
- Verify RLS policies are configured

### High-res download fails?
- Check AutoEnhance account has credits
- Verify `watermark=false` requires 1 credit
- Check user JWT is valid for RLS

---

## 📚 API Endpoints Summary

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/orders` | Create order |
| GET | `/orders` | List orders |
| GET | `/orders/{id}` | Get order details |
| DELETE | `/orders/{id}` | Delete order |
| POST | `/orders/{id}/upload` | Upload brackets |
| POST | `/orders/{id}/process` | Start processing |
| GET | `/orders/{id}/status` | Get status |
| GET | `/orders/{id}/brackets` | List uploaded brackets |
| DELETE | `/orders/{id}/brackets/{bracket_id}` | Delete bracket |
| GET | `/orders/{id}/images` | List processed images |
| POST | `/orders/{id}/images/{image_id}/download` | Download image |
| DELETE | `/orders/{id}/images/{image_id}` | Delete image |
| POST | `/webhooks/autoenhance` | AutoEnhance webhook |

---

## ✅ Success!

Your webhook-driven workflow is now complete:
- ✅ Auto-downloads previews when processing completes
- ✅ Auto-cleans up brackets after processing
- ✅ Manual delete for bad images
- ✅ Manual high-res download for good images
- ✅ Real-time notifications to iPhone
- ✅ Secure RLS-protected storage

**Next Steps:**
1. Configure webhook URL in AutoEnhance dashboard
2. Test the complete flow
3. Implement iPhone UI for preview/delete/download workflow
4. Monitor credits usage

---

**Happy Processing! 🚀**

