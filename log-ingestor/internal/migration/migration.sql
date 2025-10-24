		-- Create logs table
		CREATE UNLOGGED TABLE IF NOT EXISTS logs (
			id BIGSERIAL PRIMARY KEY,
			level TEXT NOT NULL,
			message TEXT NOT NULL,
			resource_id TEXT NOT NULL,
			timestamp TIMESTAMPTZ NOT NULL,
			trace_id TEXT,
			span_id TEXT,
			commit TEXT,
			metadata JSONB,
			indexed BOOLEAN DEFAULT FALSE,
			processing_at TIMESTAMPTZ
		);

		-- Only index fields that are actually used for filtering
		CREATE INDEX IF NOT EXISTS idx_logs_indexed ON logs(indexed);
		CREATE INDEX IF NOT EXISTS idx_logs_processing_at ON logs(processing_at);