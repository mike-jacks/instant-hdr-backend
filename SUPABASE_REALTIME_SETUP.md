# 🔧 Supabase Realtime Setup Guide

## Quick Answer: **No Setup Required!** ✅

For **broadcast events** (what we're using), **no Supabase dashboard configuration is needed**. The REST API works out of the box!

---

## What We're Using

We're using **Supabase Realtime Broadcast** via the REST API:
- **Endpoint**: `POST /realtime/v1/api/broadcast`
- **Authentication**: Publishable key (in `apikey` header)
- **Channel Format**: `order:{order_id}` (e.g., `order:550e8400-...`)

This works **immediately** - no configuration needed!

---

## Optional: Private Channels with RLS

If you want to use **private channels** (with Row Level Security), you would need to:

### 1. Create RLS Policy (Optional)

Only needed if you want to restrict who can receive broadcasts:

```sql
-- Allow authenticated users to receive broadcasts
CREATE POLICY "authenticated can receive broadcasts"
ON "realtime"."messages"
FOR SELECT
TO authenticated
USING ( true );
```

**Note**: We're using **public channels** (`order:{order_id}`), so this is **NOT required**.

---

## What's NOT Needed

❌ **Database Replication**: Not needed for broadcast (only for database change events)  
❌ **Table Replication**: Not needed for broadcast  
❌ **REPLICA IDENTITY**: Not needed for broadcast  
❌ **Dashboard Settings**: No special Realtime settings needed  
❌ **RLS Policies**: Not needed for public broadcast channels  

---

## How It Works

### Backend (Your Go Server)
```go
// Publishes to: order:{order_id}
realtimeClient.PublishOrderEvent(orderID, "download_ready", payload)
```

### Frontend (iOS App)
```swift
// Subscribes to: order:{order_id}
let channel = supabase.realtime.channel("order:\(orderId)")
channel.on("broadcast", filter: ["event": "download_ready"]) { ... }
channel.subscribe()
```

**That's it!** No Supabase dashboard configuration needed.

---

## Testing

### 1. Test Broadcast via cURL

```bash
curl -X POST "https://YOUR_PROJECT.supabase.co/realtime/v1/api/broadcast" \
  -H "apikey: YOUR_PUBLISHABLE_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{
      "topic": "order:test-order-123",
      "event": "test_event",
      "payload": { "message": "Hello from backend!" }
    }]
  }'
```

### 2. Subscribe in iOS App

```swift
let channel = supabase.realtime.channel("order:test-order-123")
channel.on("broadcast", filter: ["event": "test_event"]) { message in
    print("Received: \(message.payload)")
}
channel.subscribe()
```

If you see the message in your iOS app, **everything is working!**

---

## When Would You Need Setup?

You'd only need Supabase dashboard configuration if you wanted:

1. **Database Change Events** (not broadcast):
   - Enable Realtime replication on specific tables
   - Set `REPLICA IDENTITY FULL` on tables
   - Configure in Dashboard → Database → Replication

2. **Private Channels with RLS**:
   - Create RLS policies on `realtime.messages` table
   - Use `private: true` in channel config

3. **Presence Tracking**:
   - Configure presence policies
   - Set up presence keys

**None of these apply to our use case!** We're using simple public broadcast channels.

---

## Summary

✅ **Broadcast via REST API**: Works immediately, no setup  
✅ **Public Channels**: No RLS needed  
✅ **Publishable Key**: Sufficient for authentication  
❌ **No Dashboard Configuration**: Required  
❌ **No Database Setup**: Required  

**You're ready to go!** Just use the API and subscribe in your iOS app.

