CREATE TABLE "articles" (
	"id" text PRIMARY KEY NOT NULL,
	"slug" text NOT NULL,
	"title" text NOT NULL,
	"excerpt" text NOT NULL,
	"cover_url" text NOT NULL,
	"category" text NOT NULL,
	"published_at" timestamp with time zone NOT NULL,
	"content" text NOT NULL,
	"related_composer_ids" jsonb DEFAULT '[]'::jsonb NOT NULL,
	"related_work_ids" jsonb DEFAULT '[]'::jsonb NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"updated_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "articles_slug_unique" UNIQUE("slug")
);
--> statement-breakpoint
CREATE TABLE "composers" (
	"id" text PRIMARY KEY NOT NULL,
	"slug" text NOT NULL,
	"name" text NOT NULL,
	"name_cn" text NOT NULL,
	"birth_year" integer NOT NULL,
	"death_year" integer,
	"country" text NOT NULL,
	"period" text NOT NULL,
	"portrait_url" text NOT NULL,
	"short_bio" text NOT NULL,
	"bio" text NOT NULL,
	"style_tags" jsonb DEFAULT '[]'::jsonb NOT NULL,
	"timeline" jsonb DEFAULT '[]'::jsonb NOT NULL,
	"starter_work_ids" jsonb DEFAULT '[]'::jsonb NOT NULL,
	"related_composer_ids" jsonb DEFAULT '[]'::jsonb NOT NULL,
	"featured" boolean DEFAULT false NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"updated_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "composers_slug_unique" UNIQUE("slug")
);
--> statement-breakpoint
CREATE TABLE "performances" (
	"id" text PRIMARY KEY NOT NULL,
	"title" text NOT NULL,
	"city" text NOT NULL,
	"venue" text NOT NULL,
	"starts_at" timestamp with time zone NOT NULL,
	"artists" jsonb DEFAULT '[]'::jsonb NOT NULL,
	"program" jsonb DEFAULT '[]'::jsonb NOT NULL,
	"ticket_url" text,
	"source_url" text NOT NULL,
	"source_name" text NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"updated_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "works" (
	"id" text PRIMARY KEY NOT NULL,
	"slug" text NOT NULL,
	"composer_id" text NOT NULL,
	"title" text NOT NULL,
	"title_cn" text NOT NULL,
	"year" integer,
	"genre" text NOT NULL,
	"period" text NOT NULL,
	"description" text NOT NULL,
	"movements" jsonb DEFAULT '[]'::jsonb NOT NULL,
	"listening_links" jsonb DEFAULT '[]'::jsonb NOT NULL,
	"featured" boolean DEFAULT false NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"updated_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "works_slug_unique" UNIQUE("slug")
);
--> statement-breakpoint
ALTER TABLE "works" ADD CONSTRAINT "works_composer_id_composers_id_fk" FOREIGN KEY ("composer_id") REFERENCES "public"."composers"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
CREATE INDEX "articles_published_at_idx" ON "articles" USING btree ("published_at");--> statement-breakpoint
CREATE INDEX "articles_category_idx" ON "articles" USING btree ("category");--> statement-breakpoint
CREATE INDEX "composers_period_idx" ON "composers" USING btree ("period");--> statement-breakpoint
CREATE INDEX "composers_country_idx" ON "composers" USING btree ("country");--> statement-breakpoint
CREATE INDEX "composers_featured_idx" ON "composers" USING btree ("featured");--> statement-breakpoint
CREATE INDEX "performances_starts_at_idx" ON "performances" USING btree ("starts_at");--> statement-breakpoint
CREATE INDEX "performances_city_idx" ON "performances" USING btree ("city");--> statement-breakpoint
CREATE INDEX "performances_venue_idx" ON "performances" USING btree ("venue");--> statement-breakpoint
CREATE INDEX "works_composer_id_idx" ON "works" USING btree ("composer_id");--> statement-breakpoint
CREATE INDEX "works_period_idx" ON "works" USING btree ("period");--> statement-breakpoint
CREATE INDEX "works_genre_idx" ON "works" USING btree ("genre");--> statement-breakpoint
CREATE INDEX "works_featured_idx" ON "works" USING btree ("featured");