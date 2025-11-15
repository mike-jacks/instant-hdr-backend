-- Migration 006: Add AutoEnhance API fields to orders table
-- This allows us to cache AutoEnhance data locally for better performance
-- and offline access

-- Step 1: Add new columns for AutoEnhance data
DO $$
BEGIN
    -- Add order name (from AutoEnhance)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'orders' 
        AND column_name = 'name'
    ) THEN
        ALTER TABLE orders ADD COLUMN name TEXT;
    END IF;

    -- Add AutoEnhance status (may differ from our internal status)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'orders' 
        AND column_name = 'autoenhance_status'
    ) THEN
        ALTER TABLE orders ADD COLUMN autoenhance_status TEXT;
    END IF;

    -- Add processing state flags
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'orders' 
        AND column_name = 'is_processing'
    ) THEN
        ALTER TABLE orders ADD COLUMN is_processing BOOLEAN DEFAULT false;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'orders' 
        AND column_name = 'is_merging'
    ) THEN
        ALTER TABLE orders ADD COLUMN is_merging BOOLEAN DEFAULT false;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'orders' 
        AND column_name = 'is_deleted'
    ) THEN
        ALTER TABLE orders ADD COLUMN is_deleted BOOLEAN DEFAULT false;
    END IF;

    -- Add total images count
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'orders' 
        AND column_name = 'total_images'
    ) THEN
        ALTER TABLE orders ADD COLUMN total_images INTEGER DEFAULT 0;
    END IF;

    -- Add AutoEnhance's last updated timestamp
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'orders' 
        AND column_name = 'autoenhance_last_updated_at'
    ) THEN
        ALTER TABLE orders ADD COLUMN autoenhance_last_updated_at TIMESTAMP;
    END IF;
END $$;

-- Step 2: Create index on autoenhance_status for filtering
CREATE INDEX IF NOT EXISTS idx_orders_autoenhance_status ON orders(autoenhance_status);

-- Step 3: Create index on is_processing for active orders queries
CREATE INDEX IF NOT EXISTS idx_orders_is_processing ON orders(is_processing);

-- Step 4: Create composite index for common queries
CREATE INDEX IF NOT EXISTS idx_orders_user_status_processing ON orders(user_id, autoenhance_status, is_processing);

