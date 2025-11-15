# 🧪 Webhook Testing Guide

## Quick Test Commands

### 1. Test Empty Body (Webhook Verification)

AutoEnhance may send an empty body to verify the endpoint is active:

```bash
curl -X POST "https://your-backend.com/api/v1/webhooks/autoenhance" \
  -H "Authorization: Bearer your-webhook-token" \
  -H "Content-Type: application/json"
```

**Expected Response:**

```json
{
  "status": "ok",
  "message": "webhook endpoint is active and ready to receive events"
}
```

---

### 2. Test `webhook_updated` Event

Sent when AutoEnhance confirms webhook configuration:

```bash
curl -X POST "https://your-backend.com/api/v1/webhooks/autoenhance" \
  -H "Authorization: Bearer your-webhook-token" \
  -H "Content-Type: application/json" \
  -d '{
    "event": "webhook_updated"
  }'
```

**Expected Response:**

```json
{
  "status": "ok",
  "message": "webhook configured"
}
```

---

### 3. Test `image_processed` Event (Single Image, Still Processing)

Simulates AutoEnhance processing one image, with more images still being processed:

```bash
curl -X POST "https://your-backend.com/api/v1/webhooks/autoenhance" \
  -H "Authorization: Bearer your-webhook-token" \
  -H "Content-Type: application/json" \
  -d '{
    "event": "image_processed",
    "image_id": "img_test123",
    "error": false,
    "order_id": "550e8400-e29b-41d4-a716-446655440000",
    "order_is_processing": true
  }'
```

**Expected Response:**

```json
{
  "status": "ok"
}
```

**What Happens:**

- ✅ Event published to realtime channel `order:550e8400-...` as `webhook_image_processed`
- ✅ Frontend receives realtime event with `order_is_processing: true`
- ❌ No preview download yet (waiting for all images)

---

### 4. Test `image_processed` Event (All Images Complete)

Simulates all images finished processing:

```bash
curl -X POST "https://your-backend.com/api/v1/webhooks/autoenhance" \
  -H "Authorization: Bearer your-webhook-token" \
  -H "Content-Type: application/json" \
  -d '{
    "event": "image_processed",
    "image_id": "img_test456",
    "error": false,
    "order_id": "550e8400-e29b-41d4-a716-446655440000",
    "order_is_processing": false
  }'
```

**Expected Response:**

```json
{
  "status": "ok"
}
```

**What Happens:**

- ✅ Event published to realtime channel
- ✅ Backend downloads preview images from AutoEnhance (watermarked, FREE)
- ✅ Previews uploaded to Supabase Storage
- ✅ Database updated with file records
- ✅ Realtime event `download_ready` published with preview URLs
- ✅ Original brackets deleted from AutoEnhance (cleanup)

---

### 5. Test Error Event

Simulates image processing failure:

```bash
curl -X POST "https://your-backend.com/api/v1/webhooks/autoenhance" \
  -H "Authorization: Bearer your-webhook-token" \
  -H "Content-Type: application/json" \
  -d '{
    "event": "image_processed",
    "image_id": "img_failed789",
    "error": true,
    "order_id": "550e8400-e29b-41d4-a716-446655440000",
    "order_is_processing": false
  }'
```

**Expected Response:**

```json
{
  "status": "ok"
}
```

**What Happens:**

- ✅ Event published to realtime channel
- ✅ Order status updated to `failed` in database
- ✅ Realtime event `processing_failed` published

---

## 🔍 Testing Authentication

### Test Missing Token (Should Fail)

```bash
curl -X POST "https://your-backend.com/api/v1/webhooks/autoenhance" \
  -H "Content-Type: application/json" \
  -d '{"event": "image_processed"}'
```

**Expected Response:**

```json
{
  "error": "missing authorization token"
}
```

---

### Test Invalid Token (Should Fail)

```bash
curl -X POST "https://your-backend.com/api/v1/webhooks/autoenhance" \
  -H "Authorization: Bearer wrong-token" \
  -H "Content-Type: application/json" \
  -d '{"event": "image_processed"}'
```

**Expected Response:**

```json
{
  "error": "invalid authorization token"
}
```

---

## 🧪 Complete Test Flow

### Step 1: Create an Order

```bash
ORDER_ID=$(curl -X POST "https://your-backend.com/api/v1/orders" \
  -H "Authorization: Bearer YOUR_USER_JWT" \
  -H "Content-Type: application/json" \
  -d '{"name": "Test Order"}' | jq -r '.order_id')

echo "Order ID: $ORDER_ID"
```

### Step 2: Upload Some Images

```bash
curl -X POST "https://your-backend.com/api/v1/orders/$ORDER_ID/upload" \
  -H "Authorization: Bearer YOUR_USER_JWT" \
  -F "images=@image1.jpg" \
  -F "images=@image2.jpg" \
  -F "images=@image3.jpg"
```

### Step 3: Process the Order

```bash
curl -X POST "https://your-backend.com/api/v1/orders/$ORDER_ID/process" \
  -H "Authorization: Bearer YOUR_USER_JWT" \
  -H "Content-Type: application/json" \
  -d '{}'
```

### Step 4: Simulate Webhook (Processing Complete)

```bash
curl -X POST "https://your-backend.com/api/v1/webhooks/autoenhance" \
  -H "Authorization: Bearer your-webhook-token" \
  -H "Content-Type: application/json" \
  -d "{
    \"event\": \"image_processed\",
    \"image_id\": \"img_complete123\",
    \"error\": false,
    \"order_id\": \"$ORDER_ID\",
    \"order_is_processing\": false
  }"
```

### Step 5: Check Results

```bash
# Check files in Supabase Storage
curl -X GET "https://your-backend.com/api/v1/orders/$ORDER_ID/files" \
  -H "Authorization: Bearer YOUR_USER_JWT"

# Check order status
curl -X GET "https://your-backend.com/api/v1/orders/$ORDER_ID/status" \
  -H "Authorization: Bearer YOUR_USER_JWT"
```

---

## 📱 Testing with iOS App

### 1. Subscribe to Realtime Events

In your iOS app, subscribe to the order channel:

```swift
let channel = supabase.realtime.channel("order:\(orderId)")

channel.on("broadcast", filter: ["event": "webhook_image_processed"]) { message in
    print("Webhook event received:", message.payload)
}

channel.on("broadcast", filter: ["event": "download_ready"]) { message in
    print("Previews ready!", message.payload)
}

channel.subscribe()
```

### 2. Send Test Webhook

Use one of the test commands above, and watch your iOS app receive the realtime events!

---

## 🐛 Debugging

### Check Backend Logs

Look for:

- `"webhook received"` - Webhook endpoint called
- `"publishing realtime event"` - Event published to Supabase
- `"downloading preview images"` - Preview download started
- `"previews ready"` - Previews uploaded to Supabase

### Check Database

```sql
-- Check order status
SELECT id, status, progress, error_message
FROM orders
WHERE id = 'your-order-id';

-- Check downloaded files
SELECT filename, storage_url, is_final, created_at
FROM order_files
WHERE order_id = 'your-order-id';
```

### Check Supabase Storage

Verify files exist:

- Path: `users/{user_id}/orders/{order_id}/preview_*.jpg`
- Should be publicly accessible if RLS allows

---

## ✅ Success Indicators

After sending a webhook with `order_is_processing: false`:

1. ✅ **Database**: Order status = `previews_ready`
2. ✅ **Database**: `order_files` table has new rows with `is_final: false`
3. ✅ **Storage**: Files exist in Supabase Storage
4. ✅ **Realtime**: Frontend receives `download_ready` event
5. ✅ **API**: `GET /orders/{order_id}/files` returns preview URLs

---

## 🔧 Troubleshooting

### Webhook Returns 400

- Check JSON format is valid
- Verify all required fields are present
- Check error message for details

### Webhook Returns 401

- Verify `Authorization` header is present
- Check token matches `AUTOENHANCE_WEBHOOK_TOKEN`
- Token is case-sensitive

### No Realtime Events

- Check `SUPABASE_PUBLISHABLE_KEY` is set
- Verify Supabase URL is correct
- Check backend logs for publish errors

### No Preview Downloads

- Verify order exists in database
- Check AutoEnhance API key is valid
- Verify images exist in AutoEnhance for that order
- Check backend logs for download errors

---

## 📝 Test Script

Save this as `test_webhook.sh`:

```bash
#!/bin/bash

WEBHOOK_URL="https://your-backend.com/api/v1/webhooks/autoenhance"
WEBHOOK_TOKEN="your-webhook-token"
ORDER_ID="550e8400-e29b-41d4-a716-446655440000"

echo "Testing webhook endpoint..."

# Test 1: Empty body
echo -e "\n1. Testing empty body (verification)..."
curl -X POST "$WEBHOOK_URL" \
  -H "Authorization: Bearer $WEBHOOK_TOKEN" \
  -H "Content-Type: application/json"

# Test 2: Complete event
echo -e "\n\n2. Testing complete processing event..."
curl -X POST "$WEBHOOK_URL" \
  -H "Authorization: Bearer $WEBHOOK_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"event\": \"image_processed\",
    \"image_id\": \"img_test_$(date +%s)\",
    \"error\": false,
    \"order_id\": \"$ORDER_ID\",
    \"order_is_processing\": false
  }"

echo -e "\n\nDone! Check your database and Supabase Storage for results."
```

Make it executable:

```bash
chmod +x test_webhook.sh
./test_webhook.sh
```
