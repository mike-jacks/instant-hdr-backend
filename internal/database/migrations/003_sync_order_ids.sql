-- Migration 003: Sync order_id and autoenhance_order_id
-- Since we now use the same ID for both our database and AutoEnhance,
-- ensure all existing records have autoenhance_order_id = id

-- Step 1: Update any existing orders where autoenhance_order_id doesn't match id
UPDATE orders
SET autoenhance_order_id = id::TEXT
WHERE autoenhance_order_id IS NULL 
   OR autoenhance_order_id != id::TEXT;

-- Step 2: Make autoenhance_order_id NOT NULL (since it should always match id)
DO $$
BEGIN
    -- First ensure all NULL values are set
    UPDATE orders SET autoenhance_order_id = id::TEXT WHERE autoenhance_order_id IS NULL;
    
    -- Then make the column NOT NULL
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'orders' 
        AND column_name = 'autoenhance_order_id'
        AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE orders ALTER COLUMN autoenhance_order_id SET NOT NULL;
    END IF;
END $$;

-- Step 3: Add a check constraint to ensure they always match
-- This prevents future inconsistencies at the database level
DO $$
BEGIN
    -- Drop existing constraint if it exists
    IF EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'orders_autoenhance_order_id_matches_id'
    ) THEN
        ALTER TABLE orders DROP CONSTRAINT orders_autoenhance_order_id_matches_id;
    END IF;
    
    -- Add constraint to ensure autoenhance_order_id always equals id
    ALTER TABLE orders 
    ADD CONSTRAINT orders_autoenhance_order_id_matches_id 
    CHECK (autoenhance_order_id = id::TEXT);
EXCEPTION
    WHEN others THEN
        -- If constraint creation fails, just log and continue
        RAISE NOTICE 'Could not add constraint: %', SQLERRM;
END $$;

