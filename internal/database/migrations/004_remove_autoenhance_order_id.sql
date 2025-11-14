-- Migration 004: Remove autoenhance_order_id column
-- Since we now use AutoEnhance's generated order_id as our primary key,
-- the autoenhance_order_id column is redundant and can be removed

-- Step 1: Drop the check constraint if it exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'orders_autoenhance_order_id_matches_id'
    ) THEN
        ALTER TABLE orders DROP CONSTRAINT orders_autoenhance_order_id_matches_id;
    END IF;
END $$;

-- Step 2: Drop the index on autoenhance_order_id if it exists
DROP INDEX IF EXISTS idx_orders_autoenhance_order_id;

-- Step 3: Drop the GetOrderByAutoEnhanceOrderID function usage (handled in code)
-- We'll update the code to use GetOrder by id instead

-- Step 4: Remove the autoenhance_order_id column
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'orders' 
        AND column_name = 'autoenhance_order_id'
    ) THEN
        ALTER TABLE orders DROP COLUMN autoenhance_order_id;
    END IF;
END $$;

