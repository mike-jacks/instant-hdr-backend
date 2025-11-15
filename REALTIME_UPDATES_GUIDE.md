# ⚡ Realtime Updates Guide - Webhook to iPhone

## How It Works: End-to-End Flow

```
┌──────────┐    ┌──────────────┐    ┌──────────┐    ┌──────────────┐
│ iPhone   │───▶│ Your Backend │───▶│AutoEnhance│───▶│   Webhook    │
│ (Upload) │    │   (API)      │    │   AI      │    │   Callback   │
└──────────┘    └──────────────┘    └──────────┘    └──────────────┘
                                                              │
                                                              ▼
┌──────────┐    ┌──────────────┐    ┌──────────┐    ┌──────────────┐
│ iPhone   │◀───│   Supabase   │◀───│ Your      │◀───│  Processing  │
│(Receives)│    │   Realtime   │    │ Backend   │    │   Complete   │
└──────────┘    └──────────────┘    └──────────┘    └──────────────┘
```

---

## 🔄 Step-by-Step Process

### 1️⃣ **User Initiates Processing**

```bash
POST /api/v1/orders/{order_id}/process
```

**Backend publishes:** `processing_started` event to Supabase Realtime

```go
// internal/handlers/process.go (line 185)
h.realtimeClient.PublishOrderEvent(orderID, "processing_started",
    supabase.ProcessingStartedPayload(orderID, ""))
```

---

### 2️⃣ **AutoEnhance AI Processes Images**

- Takes 30 seconds to 5 minutes
- Backend waits (no polling needed!)
- AutoEnhance will call your webhook when done

---

### 3️⃣ **Webhook Receives Notification**

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

**Backend receives webhook:**

```go
// internal/handlers/webhook.go (line 101-103)
if !event.OrderIsProcessing {
    // All images in order are complete
    go h.storageService.HandleProcessingCompleted(event.OrderID, event.ImageID)
}
```

---

### 4️⃣ **Backend Auto-Downloads Previews**

```go
// internal/services/storage_service.go (line 63-70)
watermark := true
preview := true
fileData, err := s.autoenhanceClient.DownloadEnhanced(image.ImageID, autoenhance.DownloadOptions{
    Format:    "jpeg",
    Preview:   &preview,   // Low-res preview
    Watermark: &watermark, // Free watermarked version
})
```

- Downloads **FREE watermarked previews** from AutoEnhance
- Uploads to Supabase Storage
- Stores metadata in database

---

### 5️⃣ **Backend Publishes Realtime Event**

```go
// internal/services/storage_service.go (line 116-117)
s.realtimeClient.PublishOrderEvent(order.ID, "download_ready",
    supabase.DownloadReadyPayload(order.ID, storageURLs))
```

**Published to Supabase Realtime channels:**

- `order:{order_id}`
- `user:{user_id}`

---

### 6️⃣ **iPhone Receives Notification**

Frontend is listening:

```javascript
const channel = supabase
  .channel(`order:${orderId}`)
  .on("broadcast", { event: "download_ready" }, (payload) => {
    // payload: { order_id, storage_urls: [...] }
    console.log("Previews ready!", payload);
    showPreviewImages(payload.storage_urls);
  })
  .subscribe();
```

---

## 📡 All Realtime Events

### Event: `upload_started`

**When:** User starts uploading files  
**Payload:**

```javascript
{
  order_id: "uuid",
  timestamp: "2025-11-15T..."
}
```

---

### Event: `upload_completed`

**When:** All files uploaded successfully  
**Payload:**

```javascript
{
  order_id: "uuid",
  total_files: 6,
  timestamp: "2025-11-15T..."
}
```

---

### Event: `processing_started`

**When:** Processing begins at AutoEnhance  
**Payload:**

```javascript
{
  order_id: "uuid",
  timestamp: "2025-11-15T..."
}
```

---

### Event: `processing_progress`

**When:** During processing (optional)  
**Payload:**

```javascript
{
  order_id: "uuid",
  progress: 50,
  total_images: 2,
  timestamp: "2025-11-15T..."
}
```

---

### Event: `processing_completed` ⭐

**When:** Processing finished, previews downloaded  
**Payload:**

```javascript
{
  order_id: "uuid",
  total_images: 2,
  timestamp: "2025-11-15T..."
}
```

---

### Event: `download_ready` 🎉

**When:** Preview images available in Supabase Storage  
**Payload:**

```javascript
{
  order_id: "uuid",
  storage_urls: [
    "https://storage.supabase.co/.../preview_img_abc1_20251115.jpg",
    "https://storage.supabase.co/.../preview_img_abc2_20251115.jpg"
  ],
  timestamp: "2025-11-15T..."
}
```

**This is the most important event** - tells iPhone previews are ready!

---

### Event: `processing_failed`

**When:** Error during processing  
**Payload:**

```javascript
{
  order_id: "uuid",
  error: "Image processing failed",
  timestamp: "2025-11-15T..."
}
```

---

## 🎨 Frontend Implementation

### React/React Native Example

```javascript
import { useEffect, useState } from "react";
import { createClient } from "@supabase/supabase-js";

const supabase = createClient(process.env.SUPABASE_URL, process.env.SUPABASE_PUBLISHABLE_KEY);

function OrderStatus({ orderId }) {
  const [status, setStatus] = useState("waiting");
  const [previewUrls, setPreviewUrls] = useState([]);
  const [progress, setProgress] = useState(0);

  useEffect(() => {
    // Subscribe to order-specific channel
    const channel = supabase
      .channel(`order:${orderId}`)
      .on("broadcast", { event: "processing_started" }, () => {
        setStatus("processing");
        setProgress(0);
      })
      .on("broadcast", { event: "processing_progress" }, (payload) => {
        setProgress(payload.progress);
      })
      .on("broadcast", { event: "download_ready" }, (payload) => {
        setStatus("previews_ready");
        setPreviewUrls(payload.storage_urls);
        setProgress(100);
      })
      .on("broadcast", { event: "processing_failed" }, (payload) => {
        setStatus("failed");
        console.error("Processing failed:", payload.error);
      })
      .subscribe();

    return () => {
      supabase.removeChannel(channel);
    };
  }, [orderId]);

  return (
    <div>
      <p>Status: {status}</p>
      <p>Progress: {progress}%</p>
      {previewUrls.map((url, i) => (
        <img key={i} src={url} alt={`Preview ${i + 1}`} />
      ))}
    </div>
  );
}
```

---

### Swift/iOS Example

```swift
import Supabase

class OrderStatusViewModel: ObservableObject {
    @Published var status: String = "waiting"
    @Published var previewUrls: [URL] = []
    @Published var progress: Double = 0.0

    let supabase: SupabaseClient
    var channel: RealtimeChannel?

    init(orderId: String) {
        supabase = SupabaseClient(
            supabaseURL: URL(string: "your-supabase-url")!,
            supabaseKey: "your-publishable-key"
        )

        // Subscribe to order channel
        channel = supabase.realtime.channel("order:\(orderId)")

        // Listen for events
        channel?.on("broadcast", filter: ["event": "processing_started"]) { [weak self] message in
            self?.status = "processing"
            self?.progress = 0
        }

        channel?.on("broadcast", filter: ["event": "processing_progress"]) { [weak self] message in
            if let progress = message.payload["progress"] as? Double {
                self?.progress = progress
            }
        }

        channel?.on("broadcast", filter: ["event": "download_ready"]) { [weak self] message in
            self?.status = "previews_ready"
            self?.progress = 100

            if let urls = message.payload["storage_urls"] as? [String] {
                self?.previewUrls = urls.compactMap { URL(string: $0) }
            }
        }

        channel?.on("broadcast", filter: ["event": "processing_failed"]) { [weak self] message in
            self?.status = "failed"
            print("Processing failed:", message.payload["error"] as? String ?? "Unknown error")
        }

        channel?.subscribe()
    }

    deinit {
        channel?.unsubscribe()
    }
}
```

---

## 🔧 Backend Implementation

### How Events Are Published

```go
// internal/supabase/realtime.go

func (r *RealtimeClient) PublishOrderEvent(orderID uuid.UUID, event string, payload map[string]interface{}) {
    // Publish to order-specific channel
    r.PublishEvent(fmt.Sprintf("order:%s", orderID.String()), event, payload)
}

func (r *RealtimeClient) PublishEvent(channel, event string, payload map[string]interface{}) {
    // Implementation uses Supabase Realtime broadcast
    // Events are published to the specified channel
    // All subscribed clients receive the event instantly
}
```

### Payload Helpers

```go
// internal/supabase/realtime.go

func ProcessingStartedPayload(orderID uuid.UUID, message string) map[string]interface{} {
    return map[string]interface{}{
        "order_id":  orderID.String(),
        "timestamp": time.Now().Format(time.RFC3339),
    }
}

func DownloadReadyPayload(orderID uuid.UUID, storageURLs []string) map[string]interface{} {
    return map[string]interface{}{
        "order_id":     orderID.String(),
        "storage_urls": storageURLs,
        "timestamp":    time.Now().Format(time.RFC3339),
    }
}
```

---

## 🚀 Setup Instructions

### 1. Configure Webhook in AutoEnhance Dashboard

```
Webhook URL: https://your-api.railway.app/api/v1/webhooks/autoenhance
Authorization: Bearer <your-webhook-token>
```

### 2. Set Environment Variables

```bash
AUTOENHANCE_WEBHOOK_TOKEN=your-secret-token
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_PUBLISHABLE_KEY=your-publishable-key
```

### 3. Enable Supabase Realtime

In Supabase Dashboard:

1. Go to **Database** → **Replication**
2. Enable Realtime for relevant tables (optional - we use broadcast, not database changes)
3. Broadcast mode works without table replication

---

## 🧪 Testing Realtime

### Test 1: Manual Webhook Call

```bash
curl -X POST https://your-api.railway.app/api/v1/webhooks/autoenhance \
  -H "Authorization: Bearer your-webhook-token" \
  -H "Content-Type: application/json" \
  -d '{
    "event": "image_processed",
    "image_id": "test-image-123",
    "error": false,
    "order_id": "550e8400-e29b-41d4-a716-446655440000",
    "order_is_processing": false
  }'
```

**Expected:** Frontend receives `download_ready` event

---

### Test 2: Full Flow

```bash
# 1. Create order
ORDER_ID=$(curl -X POST https://your-api.railway.app/api/v1/orders \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Order"}' | jq -r '.order_id')

# 2. Upload files
curl -X POST https://your-api.railway.app/api/v1/orders/$ORDER_ID/upload \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -F "files=@image1.jpg" \
  -F "files=@image2.jpg" \
  -F "files=@image3.jpg"

# 3. Start processing
curl -X POST https://your-api.railway.app/api/v1/orders/$ORDER_ID/process \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"enhance_type": "property", "bracket_grouping": "auto"}'

# 4. Wait for webhook (30-300 seconds)
# 5. Frontend receives download_ready event
```

---

## 🐛 Troubleshooting

### Event Not Received?

1. **Check webhook is configured correctly:**

   ```bash
   # Send test webhook
   curl -X POST https://your-api.railway.app/api/v1/webhooks/autoenhance \
     -H "Authorization: Bearer your-webhook-token" \
     -H "Content-Type: application/json" \
     -d '{"event":"webhook_updated"}'
   ```

2. **Check frontend subscription:**

   ```javascript
   channel.subscribe((status) => {
     console.log("Subscription status:", status);
     // Should be 'SUBSCRIBED'
   });
   ```

3. **Check backend logs:**
   ```bash
   # Look for "Publishing event" logs
   railway logs
   ```

### Wrong Channel?

Make sure you're subscribing to the correct channel format:

- ✅ `order:550e8400-e29b-41d4-a716-446655440000`
- ❌ `order-550e8400-e29b-41d4-a716-446655440000` (wrong separator)

### Events Missing?

Check the order status - webhook only fires when `order_is_processing: false`:

```bash
curl https://your-api.railway.app/api/v1/orders/$ORDER_ID/status \
  -H "Authorization: Bearer $JWT_TOKEN"
```

---

## ✅ Summary

✅ **Webhook triggers** when AutoEnhance completes processing  
✅ **Backend auto-downloads** preview images (FREE)  
✅ **Supabase Realtime** broadcasts to subscribed clients  
✅ **iPhone receives** instant notification with URLs  
✅ **No polling needed** - truly real-time

**Next Steps:**

1. Configure webhook in AutoEnhance dashboard
2. Implement frontend subscription
3. Test with a real order
4. Monitor logs for events

Happy Real-timing! ⚡
