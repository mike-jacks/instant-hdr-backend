# 📸 Upload Grouping Guide - Group Brackets at Upload Time

## Overview

You can now assign **group IDs** to images during upload, telling the backend exactly how to organize brackets for HDR processing. This is the **recommended approach** for maximum control!

---

## 🎯 Why Use Upload Grouping?

Instead of relying on sequential auto-grouping after upload, you can specify groups at upload time:

✅ **More accurate** - You know which brackets belong together  
✅ **Simpler workflow** - Upload + group in one step  
✅ **No guesswork** - Processing uses your exact grouping  
✅ **Multi-room shoots** - Group by room/location automatically  

---

## 📋 How It Works

###Step 1: Upload with Group IDs

When uploading images, pass a `groups` parameter with comma-separated group identifiers:

```bash
POST /api/v1/orders/{order_id}/upload
Content-Type: multipart/form-data

images: [file1.jpg, file2.jpg, file3.jpg, file4.jpg, file5.jpg, file6.jpg]
groups: "living-room,living-room,living-room,kitchen,kitchen,kitchen"
```

**Rules:**
- Number of group IDs **must match** number of files
- Group IDs can be any string: "room1", "shot-A", "bedroom", etc.
- Order matters - first group ID goes with first file, etc.

---

### Step 2: Process with `by_upload_group`

```bash
POST /api/v1/orders/{order_id}/process
{
  "enhance_type": "property",
  "bracket_grouping": "by_upload_group"
}
```

**Result:** 2 HDR images:
- Living room: 3 brackets merged
- Kitchen: 3 brackets merged

---

## 🏠 Real-World Example: Multi-Room Property Shoot

### Scenario:
You're shooting a 3-bedroom house with 3 brackets per room (9 images total)

### Upload:

```bash
curl -X POST https://your-api.railway.app/api/v1/orders/$ORDER_ID/upload \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -F "images=@bedroom1_dark.jpg" \
  -F "images=@bedroom1_normal.jpg" \
  -F "images=@bedroom1_bright.jpg" \
  -F "images=@kitchen_dark.jpg" \
  -F "images=@kitchen_normal.jpg" \
  -F "images=@kitchen_bright.jpg" \
  -F "images=@living_dark.jpg" \
  -F "images=@living_normal.jpg" \
  -F "images=@living_bright.jpg" \
  -F "groups=bedroom1,bedroom1,bedroom1,kitchen,kitchen,kitchen,living-room,living-room,living-room"
```

### Process:

```bash
curl -X POST https://your-api.railway.app/api/v1/orders/$ORDER_ID/process \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "enhance_type": "property",
    "bracket_grouping": "by_upload_group",
    "sky_replacement": true,
    "vertical_correction": true
  }'
```

**Result:** 3 HDR images (one per room) ✅

---

## 📱 iPhone Implementation Example

### Swift Code:

```swift
func uploadImages(orderID: String, images: [UIImage], groups: [String]) async throws {
    let url = URL(string: "\(baseURL)/orders/\(orderID)/upload")!
    var request = URLRequest(url: url)
    request.httpMethod = "POST"
    request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
    
    let boundary = UUID().uuidString
    request.setValue("multipart/form-data; boundary=\(boundary)", forHTTPHeaderField: "Content-Type")
    
    var body = Data()
    
    // Add images
    for (index, image) in images.enumerated() {
        if let imageData = image.jpegData(compressionQuality: 0.9) {
            body.append("--\(boundary)\r\n")
            body.append("Content-Disposition: form-data; name=\"images\"; filename=\"image_\(index).jpg\"\r\n")
            body.append("Content-Type: image/jpeg\r\n\r\n")
            body.append(imageData)
            body.append("\r\n")
        }
    }
    
    // Add groups (comma-separated)
    let groupsString = groups.joined(separator: ",")
    body.append("--\(boundary)\r\n")
    body.append("Content-Disposition: form-data; name=\"groups\"\r\n\r\n")
    body.append(groupsString)
    body.append("\r\n")
    
    body.append("--\(boundary)--\r\n")
    
    request.httpBody = body
    
    let (data, _) = try await URLSession.shared.data(for: request)
    // Handle response
}

// Usage:
let images = [bedroom1_dark, bedroom1_normal, bedroom1_bright,
              kitchen_dark, kitchen_normal, kitchen_bright]
let groups = ["bedroom1", "bedroom1", "bedroom1",
              "kitchen", "kitchen", "kitchen"]

try await uploadImages(orderID: orderID, images: images, groups: groups)
```

---

## 🔄 Complete Workflow Example

### 1. Create Order

```bash
ORDER_ID=$(curl -X POST https://your-api.railway.app/api/v1/orders \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Property - 789 Oak Ave"}' | jq -r '.order_id')
```

### 2. Upload with Groups

```bash
curl -X POST https://your-api.railway.app/api/v1/orders/$ORDER_ID/upload \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -F "images=@img1.jpg" \
  -F "images=@img2.jpg" \
  -F "images=@img3.jpg" \
  -F "images=@img4.jpg" \
  -F "images=@img5.jpg" \
  -F "images=@img6.jpg" \
  -F "groups=room1,room1,room1,room2,room2,room2"
```

### 3. Verify Groups Were Stored

```bash
curl https://your-api.railway.app/api/v1/orders/$ORDER_ID/brackets \
  -H "Authorization: Bearer $JWT_TOKEN" | jq '.brackets[].metadata.group_id'
```

**Output:**
```json
"room1"
"room1"
"room1"
"room2"
"room2"
"room2"
```

### 4. Process with Upload Grouping

```bash
curl -X POST https://your-api.railway.app/api/v1/orders/$ORDER_ID/process \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "enhance_type": "property",
    "bracket_grouping": "by_upload_group"
  }'
```

**Response:**
```json
{
  "order_id": "...",
  "status": "processing",
  "message": "Order processing started successfully - Creating 2 HDR image(s) from 6 bracket(s)",
  "processing_params": {
    "total_brackets": 6,
    "total_images": 2
  }
}
```

---

## 🎛️ All Grouping Strategies Comparison

| Strategy | When to Use | Example Result |
|----------|-------------|----------------|
| **`by_upload_group`** ⭐ | Multi-room shoots, precise control | Groups match upload groups |
| `auto` | Simple sequential shoots | Groups by sets of 3 |
| `all` | Extreme HDR, all brackets together | 1 mega-HDR image |
| `individual` | No HDR merging needed | 1 image per bracket |
| Custom array | Advanced custom grouping | Exactly as specified |

---

## 💡 Advanced Tips

### Mixed Grouping

You can upload some brackets with groups and some without:

```bash
groups: "room1,room1,room1,,,,"
```

- First 3 images → "room1" group (3-bracket HDR)
- Last 3 images → No group → Auto-grouped into sets of 3

### Same Group, Multiple Uploads

You can upload to the same group across multiple upload calls:

**Upload 1:**
```bash
images: [room1_dark.jpg, room1_normal.jpg]
groups: "room1,room1"
```

**Upload 2:**
```bash
images: [room1_bright.jpg]
groups: "room1"
```

**Result:** All 3 brackets merge into one "room1" HDR image ✅

### Ungrouped Fallback

If `bracket_grouping` is `by_upload_group` but some brackets have no `group_id`:
- Grouped brackets → Processed by group
- Ungrouped brackets → Auto-grouped by sets of 3

**This ensures nothing is left behind!**

---

## 🐛 Error Handling

### Groups Count Mismatch

```json
{
  "error": "groups count mismatch",
  "message": "provided 4 group identifiers but 6 files"
}
```

**Fix:** Ensure `groups` has exactly as many comma-separated values as files.

### Invalid Process Strategy

If you use `by_upload_group` but no brackets have `group_id`:
- Backend **automatically falls back** to `auto` mode
- No error thrown

---

## 📊 How Groups Are Stored

Groups are stored in the `brackets` table `metadata` JSON field:

```json
{
  "group_id": "living-room",
  "CameraMake": "Apple",
  "CameraModel": "iPhone 16 Pro Max",
  "FileSize": 2823610,
  "ImageHeight": 4032,
  "ImageWidth": 3024,
  "MIMEType": "image/jpeg"
}
```

- `group_id`: Your custom grouping identifier
- Other metadata: From AutoEnhance AI (camera info, etc.)

---

## ✅ Best Practices

### 1. **Use Descriptive Group Names**
✅ Good: `"master-bedroom"`, `"kitchen-sunset"`, `"exterior-front"`  
❌ Bad: `"a"`, `"1"`, `"x"`

### 2. **Consistent Naming**
If shooting multiple properties, use a pattern:
- `"property1-living"`, `"property1-kitchen"`
- `"property2-living"`, `"property2-kitchen"`

### 3. **Default to `by_upload_group`**
```json
{
  "bracket_grouping": "by_upload_group"
}
```
This is now the **recommended default** and works even if some brackets aren't grouped.

### 4. **Validate Before Upload**
Ensure your group array length matches file count:
```javascript
if (groups.length !== files.length) {
  throw new Error('Groups must match file count')
}
```

---

## 🎯 Summary

### ✅ What You Can Do Now:

1. **Assign group IDs during upload** - One API call, done!
2. **Process with `by_upload_group`** - Uses your exact grouping
3. **Multi-room shoots made easy** - Group by room automatically
4. **No guesswork** - You control which brackets merge together
5. **Fallback to auto** - Ungrouped brackets still processed

### 🚀 Recommended Workflow:

```bash
# 1. Upload with groups
POST /orders/{id}/upload
  images: [...]
  groups: "room1,room1,room1,room2,room2,room2"

# 2. Process using upload groups
POST /orders/{id}/process
  bracket_grouping: "by_upload_group"

# 3. Wait for webhook
# 4. Review previews
# 5. Download high-res
```

---

Happy Grouping! 📸✨

