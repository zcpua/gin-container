ALTER TABLE "performances" ADD COLUMN "image_url" text;--> statement-breakpoint
ALTER TABLE "performances" ADD COLUMN "price_label" text;--> statement-breakpoint
ALTER TABLE "performances" ADD COLUMN "sale_status" text;--> statement-breakpoint
ALTER TABLE "performances" ADD COLUMN "address" text;--> statement-breakpoint
ALTER TABLE "performances" ADD COLUMN "intro" text;--> statement-breakpoint
ALTER TABLE "performances" ADD COLUMN "is_classical" boolean;--> statement-breakpoint
ALTER TABLE "performances" ADD COLUMN "source_id" text;--> statement-breakpoint
ALTER TABLE "performances" ADD COLUMN "source_metadata" jsonb;--> statement-breakpoint
CREATE UNIQUE INDEX "performances_source_id_unique" ON "performances" USING btree ("source_id");