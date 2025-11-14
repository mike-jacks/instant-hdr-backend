-- Migration 002: Migrate from Imagen to AutoEnhance AI (rename projects to orders)

-- Check if orders table already exists (from fresh install)
DO $$
BEGIN
    -- If orders table doesn't exist, create it (for fresh installs)
    IF NOT EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'orders') THEN
        -- Create orders table if it doesn't exist (fresh install)
        CREATE TABLE IF NOT EXISTS orders (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            user_id UUID NOT NULL,
            autoenhance_order_id TEXT,
            status TEXT NOT NULL DEFAULT 'created',
            progress INTEGER DEFAULT 0,
            metadata JSONB DEFAULT '{}',
            error_message TEXT,
            created_at TIMESTAMP DEFAULT NOW(),
            updated_at TIMESTAMP DEFAULT NOW()
        );
        
        -- Create order_files table if it doesn't exist
        CREATE TABLE IF NOT EXISTS order_files (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
            user_id UUID NOT NULL,
            filename TEXT NOT NULL,
            autoenhance_image_id TEXT,
            storage_path TEXT NOT NULL,
            storage_url TEXT NOT NULL,
            file_size BIGINT,
            mime_type TEXT DEFAULT 'image/jpeg',
            is_final BOOLEAN DEFAULT false,
            created_at TIMESTAMP DEFAULT NOW()
        );
    ELSE
        -- Orders table exists, so we're migrating from projects
        -- Step 1: Rename project_files to order_files first (to avoid FK issues)
        IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'project_files') THEN
            ALTER TABLE project_files RENAME TO order_files;
            ALTER TABLE order_files RENAME COLUMN project_id TO order_id;
            ALTER TABLE order_files RENAME COLUMN imagen_file_id TO autoenhance_image_id;
        END IF;
        
        -- Step 2: Rename projects to orders (if projects exists and orders doesn't)
        IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'projects') 
           AND NOT EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'orders') THEN
            ALTER TABLE projects RENAME TO orders;
        END IF;
    END IF;
END $$;

-- Step 3: Update orders table columns (only if migrating from projects)
DO $$
BEGIN
    -- Drop old Imagen columns if they exist
    IF EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'orders' AND column_name = 'imagen_project_uuid') THEN
        ALTER TABLE orders DROP COLUMN imagen_project_uuid;
    END IF;
    
    -- Add autoenhance_order_id if it doesn't exist
    IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'orders' AND column_name = 'autoenhance_order_id') THEN
        ALTER TABLE orders ADD COLUMN autoenhance_order_id TEXT;
    END IF;
    
    -- Drop unused columns if they exist
    IF EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'orders' AND column_name = 'profile_key') THEN
        ALTER TABLE orders DROP COLUMN profile_key;
    END IF;
    
    IF EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'orders' AND column_name = 'edit_id') THEN
        ALTER TABLE orders DROP COLUMN edit_id;
    END IF;
END $$;

-- Step 4: Update foreign key constraint
DO $$
DECLARE
    constraint_exists boolean;
BEGIN
    -- Check and drop old constraint on order_files if it exists
    SELECT EXISTS (
        SELECT 1 FROM pg_constraint c
        JOIN pg_class t ON c.conrelid = t.oid
        WHERE t.relname = 'order_files' 
        AND c.conname = 'project_files_project_id_fkey'
    ) INTO constraint_exists;
    
    IF constraint_exists THEN
        ALTER TABLE order_files DROP CONSTRAINT project_files_project_id_fkey;
    END IF;
    
    -- Check and drop old constraint on project_files if table and constraint exist
    IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'project_files') THEN
        SELECT EXISTS (
            SELECT 1 FROM pg_constraint c
            JOIN pg_class t ON c.conrelid = t.oid
            WHERE t.relname = 'project_files' 
            AND c.conname = 'project_files_project_id_fkey'
        ) INTO constraint_exists;
        
        IF constraint_exists THEN
            ALTER TABLE project_files DROP CONSTRAINT project_files_project_id_fkey;
        END IF;
    END IF;
END $$;

-- Add new constraint only if it doesn't exist
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'order_files') THEN
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint c
            JOIN pg_class t ON c.conrelid = t.oid
            WHERE t.relname = 'order_files' 
            AND c.conname = 'order_files_order_id_fkey'
        ) THEN
            ALTER TABLE order_files 
            ADD CONSTRAINT order_files_order_id_fkey 
            FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE;
        END IF;
    END IF;
END $$;

-- Step 5: Create brackets table
CREATE TABLE IF NOT EXISTS brackets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL,
    bracket_id TEXT NOT NULL,
    image_id TEXT,
    filename TEXT NOT NULL,
    upload_url TEXT,
    is_uploaded BOOLEAN DEFAULT false,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW()
);

-- Add foreign key constraint for brackets if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'brackets_order_id_fkey'
    ) THEN
        ALTER TABLE brackets 
        ADD CONSTRAINT brackets_order_id_fkey 
        FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE;
    END IF;
    
    -- Add unique constraint on bracket_id if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'brackets_bracket_id_key'
    ) THEN
        ALTER TABLE brackets 
        ADD CONSTRAINT brackets_bracket_id_key UNIQUE (bracket_id);
    END IF;
END $$;

-- Step 6: Update indexes
DROP INDEX IF EXISTS idx_projects_user_id;
DROP INDEX IF EXISTS idx_projects_status;
DROP INDEX IF EXISTS idx_projects_user_status;
DROP INDEX IF EXISTS idx_project_files_project_id;
DROP INDEX IF EXISTS idx_project_files_user_id;

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_user_status ON orders(user_id, status);
CREATE INDEX IF NOT EXISTS idx_orders_autoenhance_order_id ON orders(autoenhance_order_id);
CREATE INDEX IF NOT EXISTS idx_order_files_order_id ON order_files(order_id);
CREATE INDEX IF NOT EXISTS idx_order_files_user_id ON order_files(user_id);
CREATE INDEX IF NOT EXISTS idx_brackets_order_id ON brackets(order_id);
CREATE INDEX IF NOT EXISTS idx_brackets_bracket_id ON brackets(bracket_id);

-- Step 7: Update trigger
DROP TRIGGER IF EXISTS update_projects_updated_at ON orders;
CREATE TRIGGER update_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Step 8: Update Row Level Security Policies for orders
ALTER TABLE orders ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Users can select their own projects" ON orders;
DROP POLICY IF EXISTS "Users can insert their own projects" ON orders;
DROP POLICY IF EXISTS "Users can update their own projects" ON orders;
DROP POLICY IF EXISTS "Users can delete their own projects" ON orders;

CREATE POLICY "Users can select their own orders" ON orders
    FOR SELECT
    USING (auth.uid() = user_id);

CREATE POLICY "Users can insert their own orders" ON orders
    FOR INSERT
    WITH CHECK (auth.uid() = user_id);

CREATE POLICY "Users can update their own orders" ON orders
    FOR UPDATE
    USING (auth.uid() = user_id);

CREATE POLICY "Users can delete their own orders" ON orders
    FOR DELETE
    USING (auth.uid() = user_id);

-- Step 9: Update Row Level Security Policies for order_files
ALTER TABLE order_files ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Users can select their own files" ON order_files;
DROP POLICY IF EXISTS "Users can insert their own files" ON order_files;
DROP POLICY IF EXISTS "Users can update their own files" ON order_files;
DROP POLICY IF EXISTS "Users can delete their own files" ON order_files;

CREATE POLICY "Users can select their own files" ON order_files
    FOR SELECT
    USING (auth.uid() = user_id);

CREATE POLICY "Users can insert their own files" ON order_files
    FOR INSERT
    WITH CHECK (auth.uid() = user_id);

CREATE POLICY "Users can update their own files" ON order_files
    FOR UPDATE
    USING (auth.uid() = user_id);

CREATE POLICY "Users can delete their own files" ON order_files
    FOR DELETE
    USING (auth.uid() = user_id);

-- Step 10: Row Level Security Policies for brackets
ALTER TABLE brackets ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS "Users can select their own brackets" ON brackets;
DROP POLICY IF EXISTS "Users can insert their own brackets" ON brackets;
DROP POLICY IF EXISTS "Users can update their own brackets" ON brackets;
DROP POLICY IF EXISTS "Users can delete their own brackets" ON brackets;

-- Note: Brackets are accessed via order_id, so we need to check through orders table
CREATE POLICY "Users can select their own brackets" ON brackets
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM orders 
            WHERE orders.id = brackets.order_id 
            AND orders.user_id = auth.uid()
        )
    );

CREATE POLICY "Users can insert their own brackets" ON brackets
    FOR INSERT
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM orders 
            WHERE orders.id = brackets.order_id 
            AND orders.user_id = auth.uid()
        )
    );

CREATE POLICY "Users can update their own brackets" ON brackets
    FOR UPDATE
    USING (
        EXISTS (
            SELECT 1 FROM orders 
            WHERE orders.id = brackets.order_id 
            AND orders.user_id = auth.uid()
        )
    );

CREATE POLICY "Users can delete their own brackets" ON brackets
    FOR DELETE
    USING (
        EXISTS (
            SELECT 1 FROM orders 
            WHERE orders.id = brackets.order_id 
            AND orders.user_id = auth.uid()
        )
    );

