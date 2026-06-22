CREATE TABLE "users" (
	"openid" text PRIMARY KEY NOT NULL,
	"unionid" text,
	"nickname" text,
	"avatar_url" text,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"updated_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "favorites" (
	"openid" text NOT NULL,
	"performance_id" text NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "favorites_openid_performance_id_pk" PRIMARY KEY("openid","performance_id")
);
--> statement-breakpoint
CREATE TABLE "tickets" (
	"openid" text NOT NULL,
	"performance_id" text NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "tickets_openid_performance_id_pk" PRIMARY KEY("openid","performance_id")
);
--> statement-breakpoint
ALTER TABLE "favorites" ADD CONSTRAINT "favorites_openid_users_openid_fk" FOREIGN KEY ("openid") REFERENCES "public"."users"("openid") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "favorites" ADD CONSTRAINT "favorites_performance_id_performances_id_fk" FOREIGN KEY ("performance_id") REFERENCES "public"."performances"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "tickets" ADD CONSTRAINT "tickets_openid_users_openid_fk" FOREIGN KEY ("openid") REFERENCES "public"."users"("openid") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "tickets" ADD CONSTRAINT "tickets_performance_id_performances_id_fk" FOREIGN KEY ("performance_id") REFERENCES "public"."performances"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
CREATE INDEX "favorites_openid_idx" ON "favorites" USING btree ("openid");--> statement-breakpoint
CREATE INDEX "tickets_openid_idx" ON "tickets" USING btree ("openid");
