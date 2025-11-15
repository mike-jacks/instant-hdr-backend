# 📱 iOS Realtime Events Guide

## Complete Setup for iPhone App

This guide shows you how to subscribe to all realtime events from your backend using Supabase Swift client.

---

## 🚀 Setup

### 1. Add Supabase Swift to Your Project

Add to your `Package.swift` or Xcode Package Dependencies:

```swift
dependencies: [
    .package(url: "https://github.com/supabase/supabase-swift", from: "2.0.0")
]
```

Or via Swift Package Manager in Xcode:

- File → Add Packages → `https://github.com/supabase/supabase-swift`

### 2. Initialize Supabase Client

```swift
import Supabase
import Foundation

class SupabaseManager {
    static let shared = SupabaseManager()

    let client: SupabaseClient

    private init() {
        let supabaseURL = URL(string: "YOUR_SUPABASE_URL")!
        let supabaseKey = "YOUR_PUBLISHABLE_KEY"

        client = SupabaseClient(
            supabaseURL: supabaseURL,
            supabaseKey: supabaseKey
        )
    }
}
```

---

## 📡 Subscribe to Order Events

### Complete Example: OrderStatusViewModel

```swift
import SwiftUI
import Supabase
import Combine

class OrderStatusViewModel: ObservableObject {
    @Published var status: String = "waiting"
    @Published var progress: Double = 0.0
    @Published var previewURLs: [URL] = []
    @Published var errorMessage: String?
    @Published var uploadedFilesCount: Int = 0
    @Published var processedImagesCount: Int = 0
    @Published var isProcessing: Bool = false

    private let orderId: String
    private var channel: RealtimeChannel?
    private var cancellables = Set<AnyCancellable>()

    init(orderId: String) {
        self.orderId = orderId
        subscribeToEvents()
    }

    deinit {
        unsubscribe()
    }

    // MARK: - Subscription

    func subscribeToEvents() {
        let supabase = SupabaseManager.shared.client

        // Create channel for this specific order
        channel = supabase.realtime.channel("order:\(orderId)")

        // MARK: Upload Events

        // Event: upload_started
        channel?.on("broadcast", filter: ["event": "upload_started"]) { [weak self] message in
            guard let self = self,
                  let payload = message.payload as? [String: Any] else { return }

            DispatchQueue.main.async {
                self.status = "uploading"
                self.progress = 0.0
                print("📤 Upload started for order: \(payload["order_id"] ?? "")")
            }
        }

        // Event: upload_completed
        channel?.on("broadcast", filter: ["event": "upload_completed"]) { [weak self] message in
            guard let self = self,
                  let payload = message.payload as? [String: Any] else { return }

            DispatchQueue.main.async {
                self.status = "uploaded"
                self.progress = 0.25
                if let fileCount = payload["file_count"] as? Int {
                    self.uploadedFilesCount = fileCount
                }
                print("✅ Upload completed: \(self.uploadedFilesCount) files")
            }
        }

        // MARK: Processing Events

        // Event: processing_started
        channel?.on("broadcast", filter: ["event": "processing_started"]) { [weak self] message in
            guard let self = self,
                  let payload = message.payload as? [String: Any] else { return }

            DispatchQueue.main.async {
                self.status = "processing"
                self.progress = 0.5
                self.isProcessing = true
                print("⚙️ Processing started for order: \(payload["order_id"] ?? "")")
            }
        }

        // Event: webhook_image_processed (NEW - Every image as it completes)
        channel?.on("broadcast", filter: ["event": "webhook_image_processed"]) { [weak self] message in
            guard let self = self,
                  let payload = message.payload as? [String: Any] else { return }

            DispatchQueue.main.async {
                let imageId = payload["image_id"] as? String ?? ""
                let isProcessing = payload["order_is_processing"] as? Bool ?? true
                let hasError = payload["error"] as? Bool ?? false

                if hasError {
                    self.errorMessage = "Image processing failed: \(imageId)"
                    print("❌ Image processing failed: \(imageId)")
                } else {
                    self.processedImagesCount += 1
                    print("✅ Image processed: \(imageId) (\(self.processedImagesCount) total)")

                    if !isProcessing {
                        // All images done!
                        print("🎉 All images processed!")
                    }
                }
            }
        }

        // Event: download_ready (Previews available!)
        channel?.on("broadcast", filter: ["event": "download_ready"]) { [weak self] message in
            guard let self = self,
                  let payload = message.payload as? [String: Any] else { return }

            DispatchQueue.main.async {
                self.status = "previews_ready"
                self.progress = 1.0
                self.isProcessing = false

                // Extract preview URLs
                if let storageURLs = payload["storage_urls"] as? [String] {
                    self.previewURLs = storageURLs.compactMap { URL(string: $0) }
                    print("🎉 Previews ready! \(self.previewURLs.count) images available")
                }
            }
        }

        // Event: processing_failed
        channel?.on("broadcast", filter: ["event": "processing_failed"]) { [weak self] message in
            guard let self = self,
                  let payload = message.payload as? [String: Any] else { return }

            DispatchQueue.main.async {
                self.status = "failed"
                self.isProcessing = false
                self.errorMessage = payload["error"] as? String ?? "Processing failed"
                print("❌ Processing failed: \(self.errorMessage ?? "")")
            }
        }

        // Subscribe to the channel
        channel?.subscribe { [weak self] status, error in
            if let error = error {
                print("❌ Subscription error: \(error.localizedDescription)")
                DispatchQueue.main.async {
                    self?.errorMessage = "Failed to subscribe: \(error.localizedDescription)"
                }
            } else if status == .subscribed {
                print("✅ Subscribed to order:\(self?.orderId ?? "")")
            }
        }
    }

    func unsubscribe() {
        channel?.unsubscribe()
        channel = nil
    }
}
```

---

## 🎨 SwiftUI View Example

```swift
import SwiftUI

struct OrderStatusView: View {
    @StateObject private var viewModel: OrderStatusViewModel
    let orderId: String

    init(orderId: String) {
        self.orderId = orderId
        _viewModel = StateObject(wrappedValue: OrderStatusViewModel(orderId: orderId))
    }

    var body: some View {
        VStack(spacing: 20) {
            // Status Header
            VStack {
                Text("Order Status")
                    .font(.title2)
                    .bold()

                Text(viewModel.status.uppercased())
                    .font(.headline)
                    .foregroundColor(statusColor)
                    .padding(.horizontal, 16)
                    .padding(.vertical, 8)
                    .background(statusColor.opacity(0.1))
                    .cornerRadius(8)
            }

            // Progress Bar
            ProgressView(value: viewModel.progress)
                .progressViewStyle(LinearProgressViewStyle())

            // Status Details
            VStack(alignment: .leading, spacing: 8) {
                if viewModel.status == "uploading" || viewModel.status == "uploaded" {
                    HStack {
                        Image(systemName: "arrow.up.circle.fill")
                        Text("\(viewModel.uploadedFilesCount) files uploaded")
                    }
                }

                if viewModel.isProcessing {
                    HStack {
                        Image(systemName: "gearshape.fill")
                        Text("Processing \(viewModel.processedImagesCount) images...")
                    }
                }

                if viewModel.status == "previews_ready" {
                    HStack {
                        Image(systemName: "checkmark.circle.fill")
                            .foregroundColor(.green)
                        Text("\(viewModel.previewURLs.count) previews ready!")
                    }
                }
            }
            .font(.subheadline)
            .foregroundColor(.secondary)

            // Preview Images
            if !viewModel.previewURLs.isEmpty {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 12) {
                        ForEach(viewModel.previewURLs, id: \.self) { url in
                            AsyncImage(url: url) { image in
                                image
                                    .resizable()
                                    .aspectRatio(contentMode: .fill)
                            } placeholder: {
                                ProgressView()
                            }
                            .frame(width: 150, height: 150)
                            .cornerRadius(12)
                            .shadow(radius: 4)
                        }
                    }
                    .padding(.horizontal)
                }
            }

            // Error Message
            if let error = viewModel.errorMessage {
                Text(error)
                    .font(.caption)
                    .foregroundColor(.red)
                    .padding()
                    .background(Color.red.opacity(0.1))
                    .cornerRadius(8)
            }
        }
        .padding()
    }

    private var statusColor: Color {
        switch viewModel.status {
        case "uploading", "processing":
            return .blue
        case "uploaded":
            return .orange
        case "previews_ready":
            return .green
        case "failed":
            return .red
        default:
            return .gray
        }
    }
}
```

---

## 📋 All Available Events

| Event Name                | When Fired                                  | Payload                                                                  |
| ------------------------- | ------------------------------------------- | ------------------------------------------------------------------------ |
| `upload_started`          | User starts uploading files                 | `{ order_id, status: "uploading", file_count, timestamp }`               |
| `upload_completed`        | All files uploaded successfully             | `{ order_id, status: "uploaded", file_count, timestamp }`                |
| `processing_started`      | Processing begins at AutoEnhance            | `{ order_id, status: "processing", timestamp }`                          |
| `webhook_image_processed` | ⭐ **Every image processed** (from webhook) | `{ order_id, image_id, error, order_is_processing, timestamp }`          |
| `download_ready`          | 🎉 **Previews available in Supabase**       | `{ order_id, status: "previews_ready", storage_urls: [...], timestamp }` |
| `processing_failed`       | Error during processing                     | `{ order_id, status: "failed", error: "...", timestamp }`                |

---

## 🔑 Key Points

1. **Channel Name Format**: `order:{order_id}` (e.g., `order:550e8400-e29b-41d4-a716-446655440000`)

2. **Event Filtering**: Use `filter: ["event": "event_name"]` to listen to specific events

3. **Payload Access**: Access payload data via `message.payload as? [String: Any]`

4. **Thread Safety**: Always update UI on `DispatchQueue.main.async`

5. **Cleanup**: Unsubscribe in `deinit` to prevent memory leaks

---

## 🧪 Testing

### Test Event Reception

```swift
// In your view model, add logging:
channel?.on("broadcast") { message in
    print("📨 Received event: \(message.event ?? "unknown")")
    print("📦 Payload: \(message.payload ?? [:])")
}
```

### Manual Test via cURL

```bash
curl -X POST "https://YOUR_PROJECT.supabase.co/realtime/v1/api/broadcast" \
  -H "apikey: YOUR_PUBLISHABLE_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{
      "topic": "order:YOUR_ORDER_ID",
      "event": "test_event",
      "payload": { "test": "data" }
    }]
  }'
```

Your iOS app should receive this test event!

---

## 🐛 Troubleshooting

### Not Receiving Events?

1. **Check Channel Name**: Must match exactly `order:{order_id}`
2. **Check Subscription Status**: Log `status` in subscribe callback
3. **Check Network**: Ensure device can reach Supabase
4. **Check API Key**: Verify publishable key is correct
5. **Check Event Name**: Event names are case-sensitive

### Connection Issues?

```swift
// Add connection status monitoring
channel?.on("system", filter: ["status": "ok"]) { message in
    print("✅ Connected to Realtime")
}

channel?.on("system", filter: ["status": "error"]) { message in
    print("❌ Realtime connection error")
}
```

---

## 📚 Additional Resources

- [Supabase Swift Documentation](https://github.com/supabase/supabase-swift)
- [Supabase Realtime Guide](https://supabase.com/docs/guides/realtime/broadcast)
- Backend Realtime Guide: `REALTIME_UPDATES_GUIDE.md`
