package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"fpl-assistant/internal/config"
	"fpl-assistant/internal/fpl"
)

const (
	slowQueryThreshold = 100 * time.Millisecond
	cacheWriteTimeout  = 3 * time.Second
	maxBatchInsert     = 50
)

type Database struct {
	SQL *sql.DB
}

// Connect initializes the MySQL pool and applies schema migrations.
// Returns (nil, nil) when the database is disabled or unreachable so callers
// can fall back to in-memory cache.
func Connect(cfg *config.Config) (*Database, error) {
	if !cfg.DBEnabled {
		log.Println("[MySQL] Database disabled via configuration (DB_ENABLED=false).")
		return nil, nil
	}

	db, err := sql.Open("mysql", cfg.MySQLDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql connection: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		log.Printf("[MySQL Warning] Could not connect to MySQL server at %s:%d: %v. Running in in-memory fallback mode.", cfg.DBHost, cfg.DBPort, err)
		return nil, nil
	}

	log.Printf("[MySQL] Connected successfully to database '%s' at %s:%d", cfg.DBName, cfg.DBHost, cfg.DBPort)

	d := &Database{SQL: db}
	if err := d.migrate(); err != nil {
		log.Printf("[MySQL Warning] Migration warning: %v", err)
	}

	return d, nil
}

func (d *Database) Close() error {
	if d == nil || d.SQL == nil {
		return nil
	}
	return d.SQL.Close()
}

func (d *Database) migrate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	queries := []string{
		`CREATE TABLE IF NOT EXISTS fpl_cache (
			cache_key VARCHAR(255) PRIMARY KEY,
			cache_value LONGTEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			INDEX idx_expires (expires_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS manager_teams (
			team_id INT PRIMARY KEY,
			manager_name VARCHAR(255) NOT NULL,
			team_name VARCHAR(255) NOT NULL,
			overall_rank INT NOT NULL DEFAULT 0,
			total_points INT NOT NULL DEFAULT 0,
			bank DECIMAL(6,2) NOT NULL DEFAULT 0.0,
			team_value DECIMAL(6,2) NOT NULL DEFAULT 0.0,
			current_gameweek INT NOT NULL DEFAULT 1,
			last_synced_at DATETIME NOT NULL,
			INDEX idx_synced (last_synced_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS availability_history (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			player_id INT NOT NULL,
			web_name VARCHAR(100) NOT NULL,
			team_short_name VARCHAR(10) NOT NULL,
			status VARCHAR(10) NOT NULL,
			chance_of_playing INT DEFAULT 100,
			severity VARCHAR(20) NOT NULL,
			news TEXT,
			logged_at DATETIME NOT NULL,
			INDEX idx_player_logged (player_id, logged_at),
			INDEX idx_logged (logged_at),
			UNIQUE KEY uk_player_status_news (player_id, status, chance_of_playing, news(120))
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS projection_snapshots (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			team_id INT NOT NULL,
			gameweek INT NOT NULL,
			formation VARCHAR(10) NOT NULL,
			captain_name VARCHAR(100) NOT NULL,
			vice_captain_name VARCHAR(100) NOT NULL,
			total_projected_xp DECIMAL(6,1) NOT NULL,
			created_at DATETIME NOT NULL,
			UNIQUE KEY uk_team_gw (team_id, gameweek)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}

	for _, q := range queries {
		if _, err := d.execLogged(ctx, q); err != nil {
			return fmt.Errorf("migration query failed: %w", err)
		}
	}

	// Idempotent upgrades for databases created before unique/composite indexes existed.
	for _, q := range []string{
		`ALTER TABLE availability_history DROP INDEX idx_player`,
		`ALTER TABLE availability_history ADD INDEX idx_player_logged (player_id, logged_at)`,
		`ALTER TABLE availability_history ADD UNIQUE KEY uk_player_status_news (player_id, status, chance_of_playing, news(120))`,
		`ALTER TABLE projection_snapshots DROP INDEX idx_team_gw`,
		`ALTER TABLE projection_snapshots ADD UNIQUE KEY uk_team_gw (team_id, gameweek)`,
	} {
		if _, err := d.SQL.ExecContext(ctx, q); err != nil {
			log.Printf("[MySQL] schema upgrade skipped: %v", err)
		}
	}

	return nil
}

func (d *Database) execLogged(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	res, err := d.SQL.ExecContext(ctx, query, args...)
	logSlowQuery("exec", query, start, err)
	return res, err
}

func (d *Database) queryRowScan(ctx context.Context, dest []interface{}, query string, args ...interface{}) error {
	start := time.Now()
	err := d.SQL.QueryRowContext(ctx, query, args...).Scan(dest...)
	logSlowQuery("query_row", query, start, err)
	return err
}

func logSlowQuery(op, query string, start time.Time, err error) {
	dur := time.Since(start)
	if dur < slowQueryThreshold {
		return
	}
	log.Printf("[SLOW QUERY] op=%s dur=%s err=%v sql=%s", op, dur, err, compactSQL(query))
}

func compactSQL(q string) string {
	s := strings.Join(strings.Fields(q), " ")
	if len(s) > 180 {
		return s[:180] + "..."
	}
	return s
}

func (d *Database) SaveManagerTeam(ctx context.Context, teamID int, managerName, teamName string, rank, points, gw int, bank, value float64) error {
	if d == nil || d.SQL == nil {
		return nil
	}

	query := `INSERT INTO manager_teams
		(team_id, manager_name, team_name, overall_rank, total_points, bank, team_value, current_gameweek, last_synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE
			manager_name = VALUES(manager_name),
			team_name = VALUES(team_name),
			overall_rank = VALUES(overall_rank),
			total_points = VALUES(total_points),
			bank = VALUES(bank),
			team_value = VALUES(team_value),
			current_gameweek = VALUES(current_gameweek),
			last_synced_at = NOW()`

	_, err := d.execLogged(ctx, query, teamID, managerName, teamName, rank, points, bank, value, gw)
	if err != nil {
		return fmt.Errorf("save_manager_team: %w", err)
	}
	return nil
}

// AvailabilityAlertRow is one availability_history write.
type AvailabilityAlertRow struct {
	PlayerID  int
	WebName   string
	TeamShort string
	Status    string
	Chance    int
	Severity  string
	News      string
}

func (d *Database) LogAvailabilityAlert(ctx context.Context, playerID int, webName, teamShort, status string, chance int, severity, news string) error {
	return d.LogAvailabilityAlerts(ctx, []AvailabilityAlertRow{{
		PlayerID:  playerID,
		WebName:   webName,
		TeamShort: teamShort,
		Status:    status,
		Chance:    chance,
		Severity:  severity,
		News:      news,
	}})
}

// LogAvailabilityAlerts batch-inserts alerts. INSERT IGNORE + unique key
// prevents duplicate rows (and write amplification) when the same news is re-logged.
func (d *Database) LogAvailabilityAlerts(ctx context.Context, rows []AvailabilityAlertRow) error {
	if d == nil || d.SQL == nil || len(rows) == 0 {
		return nil
	}

	for start := 0; start < len(rows); start += maxBatchInsert {
		end := start + maxBatchInsert
		if end > len(rows) {
			end = len(rows)
		}
		if err := d.insertAvailabilityBatch(ctx, rows[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func (d *Database) insertAvailabilityBatch(ctx context.Context, rows []AvailabilityAlertRow) error {
	var b strings.Builder
	b.WriteString(`INSERT IGNORE INTO availability_history
		(player_id, web_name, team_short_name, status, chance_of_playing, severity, news, logged_at)
		VALUES `)
	args := make([]interface{}, 0, len(rows)*7)
	for i, row := range rows {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`(?,?,?,?,?,?,?,NOW())`)
		args = append(args, row.PlayerID, row.WebName, row.TeamShort, row.Status, row.Chance, row.Severity, row.News)
	}

	_, err := d.execLogged(ctx, b.String(), args...)
	if err != nil {
		return fmt.Errorf("log_availability_alerts: %w", err)
	}
	return nil
}

func (d *Database) SaveProjectionSnapshot(ctx context.Context, teamID, gameweek int, formation, capName, viceName string, totalXP float64) error {
	if d == nil || d.SQL == nil {
		return nil
	}

	query := `INSERT INTO projection_snapshots
		(team_id, gameweek, formation, captain_name, vice_captain_name, total_projected_xp, created_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE
			formation = VALUES(formation),
			captain_name = VALUES(captain_name),
			vice_captain_name = VALUES(vice_captain_name),
			total_projected_xp = VALUES(total_projected_xp),
			created_at = NOW()`

	_, err := d.execLogged(ctx, query, teamID, gameweek, formation, capName, viceName, totalXP)
	if err != nil {
		return fmt.Errorf("save_projection_snapshot: %w", err)
	}
	return nil
}

func (d *Database) PurgeExpiredCache(ctx context.Context) error {
	if d == nil || d.SQL == nil {
		return nil
	}
	_, err := d.execLogged(ctx, `DELETE FROM fpl_cache WHERE expires_at < NOW() LIMIT 500`)
	if err != nil {
		return fmt.Errorf("purge_expired_cache: %w", err)
	}
	return nil
}

const (
	cacheKindBootstrap = "bootstrap_static"
	cacheKindEntry     = "entry"
	cacheKindPicks     = "picks"
	cacheKindFixtures  = "fixtures"
	cacheKindJSON      = "json"
)

type cacheEnvelope struct {
	Kind    string          `json:"k"`
	Payload json.RawMessage `json:"p"`
}

type cacheWrite struct {
	op    string
	key   string
	value []byte
	exp   time.Time
}

// HybridCache implements fpl.Cache using in-memory L1 and typed MySQL L2.
type HybridCache struct {
	db     *Database
	memory *fpl.MemoryCache
	writes chan cacheWrite
	stop   chan struct{}
	wg     sync.WaitGroup
}

func NewHybridCache(database *Database) *HybridCache {
	c := &HybridCache{
		db:     database,
		memory: fpl.NewMemoryCache(),
		writes: make(chan cacheWrite, 64),
		stop:   make(chan struct{}),
	}
	c.wg.Add(2)
	go c.writeLoop()
	go c.purgeLoop()
	return c
}

func (c *HybridCache) Close() {
	if c == nil {
		return
	}
	select {
	case <-c.stop:
		return
	default:
		close(c.stop)
	}
	c.wg.Wait()
	c.memory.Close()
}

func (c *HybridCache) Get(key string) (interface{}, bool) {
	if val, ok := c.memory.Get(key); ok {
		return val, true
	}

	if c.db == nil || c.db.SQL == nil {
		return nil, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var rawValue string
	var expiresAt time.Time
	query := `SELECT cache_value, expires_at FROM fpl_cache WHERE cache_key = ? AND expires_at > NOW() LIMIT 1`
	err := c.db.queryRowScan(ctx, []interface{}{&rawValue, &expiresAt}, query, key)
	if err != nil {
		return nil, false
	}

	val, ok := decodeCacheValue([]byte(rawValue))
	if !ok {
		return nil, false
	}

	remainingTTL := time.Until(expiresAt)
	if remainingTTL > 0 {
		c.memory.Set(key, val, remainingTTL)
	}
	return val, true
}

func (c *HybridCache) Set(key string, value interface{}, ttl time.Duration) {
	c.memory.Set(key, value, ttl)

	if c.db == nil || c.db.SQL == nil {
		return
	}
	data, err := encodeCacheValue(value)
	if err != nil {
		log.Printf("[HybridCache] marshal failed for key %s: %v", key, err)
		return
	}
	c.enqueue(cacheWrite{op: "set", key: key, value: data, exp: time.Now().Add(ttl)})
}

func (c *HybridCache) Delete(key string) {
	c.memory.Delete(key)
	c.enqueue(cacheWrite{op: "delete", key: key})
}

func (c *HybridCache) Clear() {
	c.memory.Clear()
	c.enqueue(cacheWrite{op: "clear"})
}

func (c *HybridCache) enqueue(job cacheWrite) {
	select {
	case <-c.stop:
		return
	case c.writes <- job:
	default:
		log.Printf("[HybridCache] write queue full, dropping %s job", job.op)
	}
}

func (c *HybridCache) writeLoop() {
	defer c.wg.Done()
	for {
		select {
		case <-c.stop:
			return
		case job := <-c.writes:
			c.applyWrite(job)
		}
	}
}

func (c *HybridCache) applyWrite(job cacheWrite) {
	if c.db == nil || c.db.SQL == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), cacheWriteTimeout)
	defer cancel()

	var err error
	switch job.op {
	case "set":
		_, err = c.db.execLogged(ctx, `INSERT INTO fpl_cache (cache_key, cache_value, expires_at)
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE cache_value = VALUES(cache_value), expires_at = VALUES(expires_at)`,
			job.key, string(job.value), job.exp)
	case "delete":
		_, err = c.db.execLogged(ctx, `DELETE FROM fpl_cache WHERE cache_key = ?`, job.key)
	case "clear":
		_, err = c.db.execLogged(ctx, `DELETE FROM fpl_cache`)
	}
	if err != nil {
		log.Printf("[HybridCache] %s failed: %v", job.op, err)
	}
}

func (c *HybridCache) purgeLoop() {
	defer c.wg.Done()
	c.purgeOnce()
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
			c.purgeOnce()
		}
	}
}

func (c *HybridCache) purgeOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := c.db.PurgeExpiredCache(ctx); err != nil {
		log.Printf("[HybridCache] purge expired cache: %v", err)
	}
}

func kindOf(value interface{}) string {
	switch value.(type) {
	case *fpl.BootstrapStatic:
		return cacheKindBootstrap
	case *fpl.Entry:
		return cacheKindEntry
	case *fpl.PicksResponse:
		return cacheKindPicks
	case []fpl.Fixture:
		return cacheKindFixtures
	default:
		return cacheKindJSON
	}
}

func encodeCacheValue(value interface{}) ([]byte, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(cacheEnvelope{Kind: kindOf(value), Payload: payload})
}

func decodeCacheValue(raw []byte) (interface{}, bool) {
	var env cacheEnvelope
	if err := json.Unmarshal(raw, &env); err != nil || env.Kind == "" || len(env.Payload) == 0 {
		return nil, false
	}

	switch env.Kind {
	case cacheKindBootstrap:
		var v fpl.BootstrapStatic
		if json.Unmarshal(env.Payload, &v) != nil {
			return nil, false
		}
		return &v, true
	case cacheKindEntry:
		var v fpl.Entry
		if json.Unmarshal(env.Payload, &v) != nil {
			return nil, false
		}
		return &v, true
	case cacheKindPicks:
		var v fpl.PicksResponse
		if json.Unmarshal(env.Payload, &v) != nil {
			return nil, false
		}
		return &v, true
	case cacheKindFixtures:
		var v []fpl.Fixture
		if json.Unmarshal(env.Payload, &v) != nil {
			return nil, false
		}
		return v, true
	default:
		var v interface{}
		if json.Unmarshal(env.Payload, &v) != nil {
			return nil, false
		}
		return v, true
	}
}
