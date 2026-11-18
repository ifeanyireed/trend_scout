package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var DB *sql.DB
var StorageRoot string

func LoadEnvFile() {
	cwd, _ := os.Getwd()
	candidates := []string{
		".env",
		"../.env",
		"../../.env",
		filepath.Join(cwd, ".env"),
		filepath.Join(cwd, "..", ".env"),
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					k := strings.TrimSpace(parts[0])
					v := strings.TrimSpace(parts[1])
					if os.Getenv(k) == "" {
						_ = os.Setenv(k, v)
					}
				}
			}
			break
		}
	}
}

func InitStorageDirs() error {
	StorageRoot = strings.TrimSpace(os.Getenv("STORAGE_DIR"))
	if StorageRoot == "" {
		if os.Getenv("VERCEL") != "" {
			StorageRoot = "/tmp/storage"
		} else {
			cwd, err := os.Getwd()
			if err == nil {
				// Check if we are inside backend/ or root
				if _, errS := os.Stat(filepath.Join(cwd, "storage")); errS == nil {
					StorageRoot = filepath.Join(cwd, "storage")
				} else if _, errP := os.Stat(filepath.Join(cwd, "..", "storage")); errP == nil {
					StorageRoot = filepath.Clean(filepath.Join(cwd, "..", "storage"))
				} else {
					StorageRoot = filepath.Join(cwd, "storage")
				}
			} else {
				StorageRoot = "./storage"
			}
		}
	}
	subdirs := []string{"images", "audio", "subtitles", "exports", "characters", "emails"}
	for _, sub := range subdirs {
		dir := filepath.Join(StorageRoot, sub)
		if err := os.MkdirAll(dir, 0755); err != nil {
			// Fallback to /tmp/storage if current directory is read-only (e.g. AWS Lambda / Vercel)
			StorageRoot = "/tmp/storage"
			_ = os.MkdirAll(filepath.Join(StorageRoot, sub), 0755)
		}
	}
	return nil
}

func ConnectAndMigrate() (*sql.DB, error) {
	LoadEnvFile()
	if err := InitStorageDirs(); err != nil {
		return nil, fmt.Errorf("init storage dirs: %w", err)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}

	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}
	conn.SetMaxOpenConns(15)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	DB = conn
	if err := RunMigrations(ctx, conn); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return conn, nil
}

func RunMigrations(ctx context.Context, conn *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	email TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	name TEXT NOT NULL,
	role TEXT NOT NULL DEFAULT 'producer',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_settings (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
	discovery_prefs_json TEXT NOT NULL,
	monitoring_prefs_json TEXT NOT NULL,
	ai_prefs_json TEXT NOT NULL,
	notification_prefs_json TEXT NOT NULL,
	production_prefs_json TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS connected_sources (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	platform TEXT NOT NULL,
	enabled BOOLEAN NOT NULL DEFAULT TRUE,
	config_json TEXT NOT NULL DEFAULT '{}',
	quota_used INTEGER NOT NULL DEFAULT 0,
	quota_limit INTEGER NOT NULL DEFAULT 1000,
	last_scan_at TIMESTAMPTZ,
	status TEXT NOT NULL DEFAULT 'ready',
	error_message TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS source_runs (
	id TEXT PRIMARY KEY,
	source_id TEXT NOT NULL REFERENCES connected_sources(id) ON DELETE CASCADE,
	status TEXT NOT NULL,
	items_found INTEGER NOT NULL DEFAULT 0,
	new_discoveries INTEGER NOT NULL DEFAULT 0,
	started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	completed_at TIMESTAMPTZ,
	error_message TEXT
);

CREATE TABLE IF NOT EXISTS source_metrics (
	id TEXT PRIMARY KEY,
	source_id TEXT NOT NULL REFERENCES connected_sources(id) ON DELETE CASCADE,
	entity_type TEXT NOT NULL,
	external_id TEXT NOT NULL,
	metric_type TEXT NOT NULL,
	metric_value DOUBLE PRECISION NOT NULL,
	metadata_json TEXT NOT NULL DEFAULT '{}',
	recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS people (
	id TEXT PRIMARY KEY,
	canonical_name TEXT NOT NULL,
	slug TEXT NOT NULL UNIQUE,
	category TEXT NOT NULL DEFAULT 'General',
	country TEXT NOT NULL DEFAULT 'Global',
	summary TEXT,
	biography TEXT,
	birth_date TEXT,
	image_url TEXT,
	underdog_classification TEXT NOT NULL DEFAULT 'Emerging underdog',
	research_status TEXT NOT NULL DEFAULT 'pending',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS person_aliases (
	id TEXT PRIMARY KEY,
	person_id TEXT NOT NULL REFERENCES people(id) ON DELETE CASCADE,
	alias_name TEXT NOT NULL,
	source_provenance TEXT NOT NULL DEFAULT 'discovery',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE(person_id, alias_name)
);

CREATE TABLE IF NOT EXISTS events (
	id TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	description TEXT,
	event_date TEXT NOT NULL,
	category TEXT NOT NULL DEFAULT 'Breakthrough',
	location TEXT,
	source_url TEXT,
	source_name TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS person_events (
	id TEXT PRIMARY KEY,
	person_id TEXT NOT NULL REFERENCES people(id) ON DELETE CASCADE,
	event_id TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
	role TEXT NOT NULL DEFAULT 'Subject',
	achievement TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE(person_id, event_id)
);

CREATE TABLE IF NOT EXISTS discoveries (
	id TEXT PRIMARY KEY,
	person_id TEXT NOT NULL REFERENCES people(id) ON DELETE CASCADE,
	event_id TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
	source_id TEXT REFERENCES connected_sources(id) ON DELETE SET NULL,
	headline TEXT NOT NULL,
	achievement TEXT NOT NULL,
	source_url TEXT NOT NULL,
	publisher TEXT NOT NULL,
	first_detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	trend_stage TEXT NOT NULL DEFAULT 'Emerging',
	independent_coverage_count INTEGER NOT NULL DEFAULT 1,
	research_status TEXT NOT NULL DEFAULT 'pending',
	status TEXT NOT NULL DEFAULT 'active',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS trend_observations (
	id TEXT PRIMARY KEY,
	discovery_id TEXT NOT NULL REFERENCES discoveries(id) ON DELETE CASCADE,
	person_id TEXT NOT NULL REFERENCES people(id) ON DELETE CASCADE,
	trend_stage TEXT NOT NULL,
	views_total BIGINT NOT NULL DEFAULT 0,
	views_per_hour DOUBLE PRECISION NOT NULL DEFAULT 0,
	views_per_day DOUBLE PRECISION NOT NULL DEFAULT 0,
	channel_relative_perf DOUBLE PRECISION NOT NULL DEFAULT 1.0,
	independent_coverage INTEGER NOT NULL DEFAULT 1,
	coverage_growth_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
	engagement_score DOUBLE PRECISION NOT NULL DEFAULT 0,
	momentum_score DOUBLE PRECISION NOT NULL DEFAULT 0,
	saturation_score DOUBLE PRECISION NOT NULL DEFAULT 0,
	observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS videos (
	id TEXT PRIMARY KEY,
	person_id TEXT NOT NULL REFERENCES people(id) ON DELETE CASCADE,
	discovery_id TEXT REFERENCES discoveries(id) ON DELETE SET NULL,
	external_video_id TEXT NOT NULL UNIQUE,
	platform TEXT NOT NULL DEFAULT 'YouTube',
	title TEXT NOT NULL,
	channel_name TEXT NOT NULL,
	channel_subscribers BIGINT NOT NULL DEFAULT 0,
	url TEXT NOT NULL,
	views BIGINT NOT NULL DEFAULT 0,
	likes BIGINT NOT NULL DEFAULT 0,
	comments BIGINT NOT NULL DEFAULT 0,
	views_per_hour DOUBLE PRECISION NOT NULL DEFAULT 0,
	views_per_day DOUBLE PRECISION NOT NULL DEFAULT 0,
	channel_relative_multiplier DOUBLE PRECISION NOT NULL DEFAULT 1.0,
	published_at TEXT NOT NULL,
	recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS social_posts (
	id TEXT PRIMARY KEY,
	person_id TEXT NOT NULL REFERENCES people(id) ON DELETE CASCADE,
	discovery_id TEXT REFERENCES discoveries(id) ON DELETE SET NULL,
	platform TEXT NOT NULL,
	external_post_id TEXT NOT NULL UNIQUE,
	author TEXT NOT NULL,
	content TEXT NOT NULL,
	url TEXT NOT NULL,
	upvotes_or_likes BIGINT NOT NULL DEFAULT 0,
	comments_count BIGINT NOT NULL DEFAULT 0,
	shares_count BIGINT NOT NULL DEFAULT 0,
	published_at TEXT NOT NULL,
	recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS news_articles (
	id TEXT PRIMARY KEY,
	person_id TEXT NOT NULL REFERENCES people(id) ON DELETE CASCADE,
	discovery_id TEXT REFERENCES discoveries(id) ON DELETE SET NULL,
	title TEXT NOT NULL,
	url TEXT NOT NULL UNIQUE,
	publisher TEXT NOT NULL,
	author TEXT,
	summary TEXT,
	published_at TEXT NOT NULL,
	retrieved_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS research_sources (
	id TEXT PRIMARY KEY,
	person_id TEXT NOT NULL REFERENCES people(id) ON DELETE CASCADE,
	url TEXT NOT NULL,
	title TEXT NOT NULL,
	publisher TEXT NOT NULL,
	published_at TEXT,
	retrieved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	source_type TEXT NOT NULL DEFAULT 'News/Biographical',
	reliability_score DOUBLE PRECISION NOT NULL DEFAULT 0.9,
	raw_excerpt TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS research_claims (
	id TEXT PRIMARY KEY,
	person_id TEXT NOT NULL REFERENCES people(id) ON DELETE CASCADE,
	source_id TEXT REFERENCES research_sources(id) ON DELETE SET NULL,
	category TEXT NOT NULL,
	claim_text TEXT NOT NULL,
	verification_status TEXT NOT NULL DEFAULT 'Verified',
	event_date TEXT,
	contradicts_claim_id TEXT,
	confidence_score DOUBLE PRECISION NOT NULL DEFAULT 0.9,
	notes TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS person_timelines (
	id TEXT PRIMARY KEY,
	person_id TEXT NOT NULL REFERENCES people(id) ON DELETE CASCADE,
	claim_id TEXT REFERENCES research_claims(id) ON DELETE SET NULL,
	source_id TEXT REFERENCES research_sources(id) ON DELETE SET NULL,
	year_or_date TEXT NOT NULL,
	chronological_order INTEGER NOT NULL,
	phase TEXT NOT NULL,
	title TEXT NOT NULL,
	description TEXT NOT NULL,
	verification_status TEXT NOT NULL DEFAULT 'Verified',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS underdog_assessments (
	id TEXT PRIMARY KEY,
	person_id TEXT NOT NULL REFERENCES people(id) ON DELETE CASCADE,
	discovery_id TEXT NOT NULL UNIQUE REFERENCES discoveries(id) ON DELETE CASCADE,
	classification TEXT NOT NULL,
	origins_summary TEXT NOT NULL,
	early_ambitions TEXT NOT NULL,
	disadvantages TEXT NOT NULL,
	rejections_and_failures TEXT NOT NULL,
	obstacles TEXT NOT NULL,
	turning_points TEXT NOT NULL,
	career_progression TEXT NOT NULL,
	recent_breakthrough TEXT NOT NULL,
	contradictions_report TEXT NOT NULL DEFAULT 'No factual contradictions detected across primary sources.',
	research_gaps_report TEXT NOT NULL DEFAULT 'Early childhood private family interviews remain open for supplemental archival discovery.',
	audience_interest_report TEXT NOT NULL DEFAULT 'High organic engagement across sports/culture forums and YouTube commentary channels.',
	assessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS opportunity_scores (
	id TEXT PRIMARY KEY,
	discovery_id TEXT NOT NULL UNIQUE REFERENCES discoveries(id) ON DELETE CASCADE,
	person_id TEXT NOT NULL REFERENCES people(id) ON DELETE CASCADE,
	overall_score DOUBLE PRECISION NOT NULL,
	underdog_strength DOUBLE PRECISION NOT NULL,
	transformation_score DOUBLE PRECISION NOT NULL,
	achievement_significance DOUBLE PRECISION NOT NULL,
	audience_momentum DOUBLE PRECISION NOT NULL,
	documentary_potential DOUBLE PRECISION NOT NULL,
	timeliness_score DOUBLE PRECISION NOT NULL,
	evidence_confidence DOUBLE PRECISION NOT NULL,
	saturation_risk DOUBLE PRECISION NOT NULL,
	score_rationale_json TEXT NOT NULL,
	calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS watchlists (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	person_id TEXT NOT NULL REFERENCES people(id) ON DELETE CASCADE,
	discovery_id TEXT NOT NULL REFERENCES discoveries(id) ON DELETE CASCADE,
	notes TEXT,
	alert_on_stage_change BOOLEAN NOT NULL DEFAULT TRUE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE(user_id, discovery_id)
);

CREATE TABLE IF NOT EXISTS notifications (
	id TEXT PRIMARY KEY,
	user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
	discovery_id TEXT REFERENCES discoveries(id) ON DELETE SET NULL,
	project_id TEXT,
	type TEXT NOT NULL,
	severity TEXT NOT NULL DEFAULT 'info',
	title TEXT NOT NULL,
	message TEXT NOT NULL,
	action_url TEXT NOT NULL,
	metadata_json TEXT NOT NULL DEFAULT '{}',
	is_read BOOLEAN NOT NULL DEFAULT FALSE,
	emailed_at TIMESTAMPTZ,
	extension_notified_at TIMESTAMPTZ,
	dedupe_key TEXT UNIQUE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS research_jobs (
	id TEXT PRIMARY KEY,
	person_id TEXT NOT NULL REFERENCES people(id) ON DELETE CASCADE,
	discovery_id TEXT REFERENCES discoveries(id) ON DELETE SET NULL,
	status TEXT NOT NULL DEFAULT 'completed',
	depth TEXT NOT NULL DEFAULT 'deep',
	sources_collected INTEGER NOT NULL DEFAULT 0,
	claims_verified INTEGER NOT NULL DEFAULT 0,
	contradictions_found INTEGER NOT NULL DEFAULT 0,
	gaps_found INTEGER NOT NULL DEFAULT 0,
	started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	completed_at TIMESTAMPTZ,
	error_message TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS documentary_projects (
	id TEXT PRIMARY KEY,
	person_id TEXT NOT NULL REFERENCES people(id) ON DELETE CASCADE,
	discovery_id TEXT REFERENCES discoveries(id) ON DELETE SET NULL,
	title TEXT NOT NULL,
	slug TEXT NOT NULL UNIQUE,
	status TEXT NOT NULL DEFAULT 'script_ready',
	target_duration_minutes INTEGER NOT NULL DEFAULT 20,
	visual_style TEXT NOT NULL DEFAULT 'Cinematic Archival Realism (35mm Documentary Stills)',
	aspect_ratio TEXT NOT NULL DEFAULT '16:9',
	image_interval_seconds INTEGER NOT NULL DEFAULT 8,
	thumbnail_concept TEXT,
	youtube_titles_json TEXT NOT NULL DEFAULT '[]',
	youtube_description TEXT,
	music_suggestions TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS documentary_scripts (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL UNIQUE REFERENCES documentary_projects(id) ON DELETE CASCADE,
	current_version INTEGER NOT NULL DEFAULT 1,
	title TEXT NOT NULL,
	opening_hook TEXT NOT NULL,
	full_narration_text TEXT NOT NULL,
	word_count INTEGER NOT NULL DEFAULT 0,
	estimated_duration_seconds INTEGER NOT NULL DEFAULT 1200,
	status TEXT NOT NULL DEFAULT 'draft',
	approved_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS script_versions (
	id TEXT PRIMARY KEY,
	script_id TEXT NOT NULL REFERENCES documentary_scripts(id) ON DELETE CASCADE,
	version_number INTEGER NOT NULL,
	sections_json TEXT NOT NULL,
	full_narration_text TEXT NOT NULL,
	word_count INTEGER NOT NULL,
	estimated_duration_seconds INTEGER NOT NULL,
	change_summary TEXT NOT NULL,
	created_by TEXT NOT NULL DEFAULT 'DocumentaryWriterAgent',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS script_references (
	id TEXT PRIMARY KEY,
	script_id TEXT NOT NULL REFERENCES documentary_scripts(id) ON DELETE CASCADE,
	section_index INTEGER NOT NULL,
	paragraph_index INTEGER NOT NULL,
	claim_id TEXT REFERENCES research_claims(id) ON DELETE SET NULL,
	source_id TEXT REFERENCES research_sources(id) ON DELETE SET NULL,
	citation_note TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS character_references (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL REFERENCES documentary_projects(id) ON DELETE CASCADE,
	person_id TEXT NOT NULL REFERENCES people(id) ON DELETE CASCADE,
	character_name TEXT NOT NULL,
	role TEXT NOT NULL,
	age_range TEXT NOT NULL,
	appearance_description TEXT NOT NULL,
	hair_description TEXT NOT NULL,
	clothing_description TEXT NOT NULL,
	body_proportions TEXT NOT NULL,
	locations_description TEXT NOT NULL,
	environmental_refs TEXT NOT NULL,
	visual_style TEXT NOT NULL,
	color_palette TEXT NOT NULL,
	master_prompt TEXT NOT NULL,
	reference_image_path TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS image_assets (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL REFERENCES documentary_projects(id) ON DELETE CASCADE,
	segment_id TEXT,
	character_ref_id TEXT,
	file_path TEXT NOT NULL,
	file_url TEXT NOT NULL,
	prompt_used TEXT NOT NULL,
	provider TEXT NOT NULL DEFAULT 'TrendScout Storyboard Engine / Google Flow Ready',
	width INTEGER NOT NULL DEFAULT 1920,
	height INTEGER NOT NULL DEFAULT 1080,
	status TEXT NOT NULL DEFAULT 'ready',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS visual_segments (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL REFERENCES documentary_projects(id) ON DELETE CASCADE,
	script_id TEXT NOT NULL REFERENCES documentary_scripts(id) ON DELETE CASCADE,
	segment_number INTEGER NOT NULL,
	segment_code TEXT NOT NULL,
	start_seconds DOUBLE PRECISION NOT NULL,
	end_seconds DOUBLE PRECISION NOT NULL,
	duration_seconds DOUBLE PRECISION NOT NULL,
	narration_passage TEXT NOT NULL,
	image_prompt TEXT NOT NULL,
	character_ref_id TEXT REFERENCES character_references(id) ON DELETE SET NULL,
	location_ref TEXT NOT NULL,
	generation_status TEXT NOT NULL DEFAULT 'prompt_ready',
	image_asset_id TEXT REFERENCES image_assets(id) ON DELETE SET NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audio_assets (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL REFERENCES documentary_projects(id) ON DELETE CASCADE,
	segment_id TEXT,
	title TEXT NOT NULL,
	file_path TEXT NOT NULL,
	file_url TEXT NOT NULL,
	provider TEXT NOT NULL DEFAULT 'TrendScout Acoustic Narration Engine',
	voice_id TEXT NOT NULL DEFAULT 'Documentary-Baritone-Narrator',
	duration_seconds DOUBLE PRECISION NOT NULL DEFAULT 0,
	transcript_text TEXT NOT NULL,
	is_master_narration BOOLEAN NOT NULL DEFAULT TRUE,
	status TEXT NOT NULL DEFAULT 'ready',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS subtitle_assets (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL REFERENCES documentary_projects(id) ON DELETE CASCADE,
	audio_asset_id TEXT REFERENCES audio_assets(id) ON DELETE SET NULL,
	format TEXT NOT NULL DEFAULT 'srt',
	file_path TEXT NOT NULL,
	file_url TEXT NOT NULL,
	cue_count INTEGER NOT NULL DEFAULT 0,
	content_text TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS production_exports (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL REFERENCES documentary_projects(id) ON DELETE CASCADE,
	export_type TEXT NOT NULL,
	file_path TEXT NOT NULL,
	file_url TEXT NOT NULL,
	file_size_bytes BIGINT NOT NULL DEFAULT 0,
	manifest_json TEXT NOT NULL DEFAULT '{}',
	video_assembled BOOLEAN NOT NULL DEFAULT FALSE,
	status TEXT NOT NULL DEFAULT 'completed',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS background_jobs (
	id TEXT PRIMARY KEY,
	job_type TEXT NOT NULL,
	queue_name TEXT NOT NULL DEFAULT 'default',
	priority INTEGER NOT NULL DEFAULT 5,
	payload_json TEXT NOT NULL DEFAULT '{}',
	idempotency_key TEXT UNIQUE,
	status TEXT NOT NULL DEFAULT 'queued',
	attempts INTEGER NOT NULL DEFAULT 0,
	max_attempts INTEGER NOT NULL DEFAULT 3,
	next_run_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	locked_by TEXT,
	locked_at TIMESTAMPTZ,
	heartbeat_at TIMESTAMPTZ,
	result_json TEXT,
	error_message TEXT,
	started_at TIMESTAMPTZ,
	completed_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audit_logs (
	id TEXT PRIMARY KEY,
	user_id TEXT,
	actor TEXT NOT NULL,
	action TEXT NOT NULL,
	entity_type TEXT NOT NULL,
	entity_id TEXT,
	details_json TEXT NOT NULL DEFAULT '{}',
	ip_address TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_discoveries_stage ON discoveries(trend_stage);
CREATE INDEX IF NOT EXISTS idx_discoveries_person ON discoveries(person_id);
CREATE INDEX IF NOT EXISTS idx_trend_obs_discovery ON trend_observations(discovery_id, observed_at);
CREATE INDEX IF NOT EXISTS idx_opp_scores_overall ON opportunity_scores(overall_score DESC);
CREATE INDEX IF NOT EXISTS idx_claims_person ON research_claims(person_id);
CREATE INDEX IF NOT EXISTS idx_timelines_person ON person_timelines(person_id, chronological_order);
CREATE INDEX IF NOT EXISTS idx_visual_segments_project ON visual_segments(project_id, segment_number);
CREATE INDEX IF NOT EXISTS idx_bg_jobs_status_next ON background_jobs(status, next_run_at, priority DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_user_read ON notifications(user_id, is_read, created_at DESC);
`
	_, err := conn.ExecContext(ctx, schema)
	return err
}
