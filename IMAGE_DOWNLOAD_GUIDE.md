# Image Download Guide

## Overview

This guide explains how to download processed images from AutoEnhance AI to Supabase Storage, making them accessible to your iPhone app.

## Workflow

```
1. Process images → 2. List images → 3. Download preview → 4. Download high-res
```

## API Endpoints

### 1. List Processed Images

**Endpoint:** `GET /api/v1/orders/{order_id}/images`

**Description:** Get all processed images for an order with their download status.

**Response:**
```json
{
  "images": [
    {
      "image_id": "img_abc123",
      "image_name": "enhanced_image_1.jpg",
      "status": "completed",
      "enhance_type": "property",
      "downloaded": false,
      "preview_downloaded": false,
      "high_res_downloaded": false,
      "preview_url": "",
      "high_res_url": "",
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

**cURL Example:**
```bash
curl -X GET \
  'https://your-backend.com/api/v1/orders/03b42480-8350-421a-b762-a259f71b0b9f/images' \
  -H 'Authorization: Bearer YOUR_JWT_TOKEN'
```

---

### 2. Download Image (Preview or High-Res)

**Endpoint:** `POST /api/v1/orders/{order_id}/images/{image_id}/download`

**Description:** Download an image from AutoEnhance and store it in Supabase Storage.

**Request Body:**
```json
{
  "quality": "preview"  // "preview" or "high"
}
```

**Response:**
```json
{
  "image_id": "img_abc123",
  "quality": "preview",
  "url": "https://your-supabase.supabase.co/storage/v1/object/public/hdr-images/users/{user_id}/orders/{order_id}/img_abc123_preview.jpg",
  "file_size": 245678,
  "message": "Image downloaded successfully to Supabase Storage (preview)"
}
```

**cURL Example (Preview):**
```bash
curl -X POST \
  'https://your-backend.com/api/v1/orders/03b42480-8350-421a-b762-a259f71b0b9f/images/img_abc123/download' \
  -H 'Authorization: Bearer YOUR_JWT_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{"quality": "preview"}'
```

**cURL Example (High-Res):**
```bash
curl -X POST \
  'https://your-backend.com/api/v1/orders/03b42480-8350-421a-b762-a259f71b0b9f/images/img_abc123/download' \
  -H 'Authorization: Bearer YOUR_JWT_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{"quality": "high"}'
```

---

## Quality Options

### Preview
- **Max Width:** 800px
- **Use Case:** Quick previews in the app, thumbnails, gallery view
- **File Size:** ~100-300KB
- **Format:** JPEG

### High
- **Resolution:** Full resolution from AutoEnhance
- **Use Case:** Final download for client delivery
- **File Size:** ~2-5MB
- **Format:** JPEG

---

## Complete Workflow Example

### Step 1: Create Order
```bash
curl -X POST 'https://your-backend.com/api/v1/orders' \
  -H 'Authorization: Bearer YOUR_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{}'
```

**Response:**
```json
{
  "order_id": "03b42480-8350-421a-b762-a259f71b0b9f",
  "status": "created"
}
```

---

### Step 2: Upload Brackets
```bash
curl -X POST 'https://your-backend.com/api/v1/orders/03b42480-8350-421a-b762-a259f71b0b9f/upload' \
  -H 'Authorization: Bearer YOUR_TOKEN' \
  -F 'files=@IMG_3090.JPG' \
  -F 'files=@IMG_3091.JPG' \
  -F 'files=@IMG_3092.JPG'
```

---

### Step 3: Process Images
```bash
curl -X POST 'https://your-backend.com/api/v1/orders/03b42480-8350-421a-b762-a259f71b0b9f/process' \
  -H 'Authorization: Bearer YOUR_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{
    "enhance_type": "property",
    "sky_replacement": true,
    "window_pull_type": "medium",
    "vertical_correction": true,
    "lens_correction": true
  }'
```

---

### Step 4: Check Processing Status
```bash
curl -X GET 'https://your-backend.com/api/v1/orders/03b42480-8350-421a-b762-a259f71b0b9f/status' \
  -H 'Authorization: Bearer YOUR_TOKEN'
```

**Response:**
```json
{
  "order_id": "03b42480-8350-421a-b762-a259f71b0b9f",
  "status": "processing",
  "autoenhance_status": "processing",
  "total_images": 1,
  "is_processing": true
}
```

**Wait until `status` is `completed` before proceeding.**

---

### Step 5: List Processed Images
```bash
curl -X GET 'https://your-backend.com/api/v1/orders/03b42480-8350-421a-b762-a259f71b0b9f/images' \
  -H 'Authorization: Bearer YOUR_TOKEN'
```

**Response:**
```json
{
  "images": [
    {
      "image_id": "img_abc123",
      "image_name": "enhanced_image_1.jpg",
      "status": "completed",
      "preview_downloaded": false,
      "high_res_downloaded": false
    }
  ]
}
```

---

### Step 6: Download Preview for iPhone
```bash
curl -X POST 'https://your-backend.com/api/v1/orders/03b42480-8350-421a-b762-a259f71b0b9f/images/img_abc123/download' \
  -H 'Authorization: Bearer YOUR_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{"quality": "preview"}'
```

**Response:**
```json
{
  "image_id": "img_abc123",
  "quality": "preview",
  "url": "https://your-supabase.supabase.co/storage/v1/object/public/hdr-images/users/{user_id}/orders/{order_id}/img_abc123_preview.jpg",
  "file_size": 245678,
  "message": "Image downloaded successfully to Supabase Storage (preview)"
}
```

**iPhone can now access the image at the returned URL.**

---

### Step 7: Download High-Res (After Preview)
```bash
curl -X POST 'https://your-backend.com/api/v1/orders/03b42480-8350-421a-b762-a259f71b0b9f/images/img_abc123/download' \
  -H 'Authorization: Bearer YOUR_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{"quality": "high"}'
```

**Response:**
```json
{
  "image_id": "img_abc123",
  "quality": "high",
  "url": "https://your-supabase.supabase.co/storage/v1/object/public/hdr-images/users/{user_id}/orders/{order_id}/img_abc123_high.jpg",
  "file_size": 3245678,
  "message": "Image downloaded successfully to Supabase Storage (high)"
}
```

---

## iPhone Implementation Tips

### 1. List Images After Processing
```swift
func fetchProcessedImages(orderId: String) async throws -> [ProcessedImage] {
    let url = URL(string: "https://your-backend.com/api/v1/orders/\(orderId)/images")!
    var request = URLRequest(url: url)
    request.setValue("Bearer \(authToken)", forHTTPHeaderField: "Authorization")
    
    let (data, _) = try await URLSession.shared.data(for: request)
    let response = try JSONDecoder().decode(ImagesResponse.self, from: data)
    return response.images
}
```

### 2. Download Preview First
```swift
func downloadPreview(orderId: String, imageId: String) async throws -> String {
    let url = URL(string: "https://your-backend.com/api/v1/orders/\(orderId)/images/\(imageId)/download")!
    var request = URLRequest(url: url)
    request.httpMethod = "POST"
    request.setValue("Bearer \(authToken)", forHTTPHeaderField: "Authorization")
    request.setValue("application/json", forHTTPHeaderField: "Content-Type")
    request.httpBody = try JSONEncoder().encode(["quality": "preview"])
    
    let (data, _) = try await URLSession.shared.data(for: request)
    let response = try JSONDecoder().decode(DownloadImageResponse.self, from: data)
    return response.url // Use this URL to display the image
}
```

### 3. Show Preview in Gallery
```swift
AsyncImage(url: URL(string: previewURL)) { image in
    image.resizable().aspectRatio(contentMode: .fit)
} placeholder: {
    ProgressView()
}
```

### 4. Download High-Res on Demand
```swift
func downloadHighRes(orderId: String, imageId: String) async throws -> String {
    // Same as downloadPreview but with "quality": "high"
    // ...
}
```

---

## Storage Structure

Files are stored in Supabase Storage with the following structure:

```
hdr-images/
└── users/
    └── {user_id}/
        └── orders/
            └── {order_id}/
                ├── img_abc123_preview.jpg
                ├── img_abc123_high.jpg
                ├── img_xyz456_preview.jpg
                └── img_xyz456_high.jpg
```

---

## Error Handling

### Image Not Ready
If processing isn't complete, you'll get:
```json
{
  "images": []
}
```

Wait and retry after a few seconds.

### Image Not Found
```json
{
  "error": "image not found",
  "message": "Image does not exist in AutoEnhance"
}
```

### Storage Error
```json
{
  "error": "failed to upload to storage",
  "message": "..."
}
```

---

## Best Practices

1. **Always download preview first** - Much faster and cheaper
2. **Only download high-res when user requests it** - Saves bandwidth and storage
3. **Check `preview_downloaded` and `high_res_downloaded` flags** - Avoid re-downloading
4. **Cache URLs on the device** - Once downloaded, URLs don't change
5. **Use Supabase Storage URLs directly** - No need to proxy through your backend

---

## Summary

✅ **List images** to see what's ready  
✅ **Download preview** for quick viewing (800px, ~200KB)  
✅ **Download high-res** when user needs full quality  
✅ **Access via Supabase URLs** - Fast and reliable  
✅ **Track download status** - Avoid redundant downloads  

Your iPhone app now has a complete image download workflow! 🎉

