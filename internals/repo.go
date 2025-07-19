package internals

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type VideoRepository struct {
	folderPath string
	db         *sql.DB
}

func NewRepository(config Config) (VideoRepository, error) {
	repo := VideoRepository{}

	db, err := openDatabase(config.Database)
	if err != nil {
		return VideoRepository{}, err
	}
	repo.db = db
	repo.folderPath = config.VideoFolder

	return repo, nil
}

func (repo VideoRepository) Close() error {
	return repo.db.Close()
}

func (repo VideoRepository) LastFolderUpdate() (*time.Time, error) {
	stmt, err := repo.db.Prepare("select last_update from video_update where id = 1")
	if err != nil {
		return nil, err
	}

	var timestamp string
	err = stmt.QueryRow().Scan(&timestamp)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	actualTime, err := time.Parse(time.DateTime, timestamp)
	if err != nil {
		return nil, err
	}

	return &actualTime, nil
}

func (repo VideoRepository) ListAllSaved() ([]Video, error) {
	return repo.queryVideos(
		`
		select
			id,
			filename,
			nickname,
			tags,
			created_at,
			status
		from
			videos
		where
			status = ?
	    order by
	      created_at
		`,
		VideoSaved,
	)
}

func (repo VideoRepository) NextInQueue(quantity int) ([]Video, error) {
	return repo.queryVideos(
		`
		select
			id,
			filename,
			nickname,
			tags,
			created_at,
			status
		from
			videos
		where
			status = ?
		order by
			created_at
		limit ?
		`,
		VideoUnwatched,
		quantity,
	)
}

func (repo VideoRepository) FindById(id int32) (*Video, error) {
	rows, err := repo.db.Query(
		`
		select
			id,
			filename,
			nickname,
			tags,
			created_at,
			status
		from
			videos
		where
			id = ?
		`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	video, err := readVideoFromRow(rows)
	if err != nil {
		return nil, err
	}

	return &video, nil
}

func (repo VideoRepository) NextSavedById(id int32) (*Video, error) {
	rows, err := repo.db.Query(
		`
		select
			id,
			filename,
			nickname,
			tags,
			created_at,
			status
		from
			videos
		where
			status = ?
	        and created_at >= (select created_at from videos where id = ?)
	        and id <> ?
	    order by
	        created_at
	    limit
	      	1
		`,
		VideoSaved,
		id,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	video, err := readVideoFromRow(rows)
	if err != nil {
		return nil, err
	}

	return &video, nil

}

func (repo VideoRepository) Update(video Video) error {
	var nickname any
	if video.Nickname.Valid && len(video.Nickname.String) > 0 {
		nickname = video.Nickname.String
	} else {
		nickname = nil
	}

	var tags any
	if len(video.Tags) > 0 {
		tags = strings.Join(video.Tags, ",")
	} else {
		tags = nil
	}

	_, err := repo.db.Exec(
		`
		update videos set
			status = ?,
			nickname = ?,
			tags = ?
		where
			id = ?
		`,
		video.Status,
		nickname,
		tags,
		video.Id,
	)

	log.Println("Updated video", video.Id)

	return err
}

func (repo VideoRepository) ImportJsonFile(path string) error {
	jsonFile, err := readVideoJsonFile(path)
	if err != nil {
		return err
	}

	tx, err := repo.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		insert into videos
			(filename, created_at, status)
		values
			(?, ?, ?)
		on conflict (filename) do nothing
	`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, entry := range jsonFile.Watched {
		_, err = stmt.Exec(entry.Name, entry.Date, StatusFromWatchedEntry(entry))
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	_, err = stmt.Exec(jsonFile.Current.Name, jsonFile.Current.Date, VideoUnwatched)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, video := range jsonFile.ToWatch {
		_, err = stmt.Exec(video.Name, video.Date, VideoUnwatched)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (repo VideoRepository) ImportFsEntries(entries []VideoFsEntry) error {
	tx, err := repo.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		insert into videos
			(filename, created_at, status)
		values
			(?, ?, ?)
		on conflict (filename) do nothing
	`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, entry := range entries {
		videoStatus := VideoUnwatched
		if entry.IsTruncated {
			videoStatus = VideoWatched
		}

		res, err := stmt.Exec(
			entry.Filename,
			entry.LastModifiedTime,
			videoStatus,
		)

		if rows, err := res.RowsAffected(); err == nil && rows > 0 {
			fmt.Println("Found", entry.Filename)
		}

		if err != nil {
			tx.Rollback()
			return err
		}
	}

	tx.Exec("update video_update set last_update = datetime('now') where id = 1;")

	return tx.Commit()
}

func (repo VideoRepository) ListDirVideos() ([]VideoFsEntry, error) {
	entries, err := os.ReadDir(repo.folderPath)
	if err != nil {
		return nil, err
	}

	files := make([]VideoFsEntry, len(entries))
	file_index := 0
	for _, entry := range entries {
		if entry.IsDir() || !hasVideoExtension(entry.Name()) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return nil, err
		}

		files[file_index] = VideoFsEntry{
			Filename:         entry.Name(),
			LastModifiedTime: info.ModTime(),
			IsTruncated:      info.Size() <= 0,
		}
		file_index += 1
	}

	files = files[:file_index]

	return files, nil
}

func (repo VideoRepository) QueryStats() (VideoStats, error) {
	rows, err := repo.db.Query(
		`
		select
			status,
			count(id)
		from
			videos
		group by
			status
		order by
			status
		`,
	)
	if err != nil {
		return VideoStats{}, err
	}
	defer rows.Close()

	stats := VideoStats{
		Unwatched: 0,
		Watched:   0,
		Liked:     0,
		Saved:     0,
	}
	for rows.Next() {
		var status VideoStatus
		var quantity int

		err = rows.Scan(&status, &quantity)
		if err != nil {
			return VideoStats{}, nil
		}

		switch status {
		case VideoUnwatched:
			stats.Unwatched = quantity
		case VideoWatched:
			stats.Watched = quantity
		case VideoLiked:
			stats.Liked = quantity
		case VideoSaved:
			stats.Saved = quantity
		}
	}

	return stats, nil
}

func (repo VideoRepository) queryVideos(sql string, args ...any) ([]Video, error) {
	rows, err := repo.db.Query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []Video
	for rows.Next() {
		video, err := readVideoFromRow(rows)
		if err != nil {
			return nil, err
		}
		videos = append(videos, video)
	}

	return videos, nil
}

func openDatabase(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		create table if not exists videos (
			id integer primary key,
			filename text not null unique,
			created_at datetime not null,
			status integer
		);

		create table if not exists migrations (
			id integer primary key,
			version integer not null
		);

		insert into migrations(id, version) values (1, 0)
		on conflict(id) do nothing;
	`)

	if err != nil {
		db.Close()
		return nil, err
	}
	log.Println("Database opened")

	if err = executeMigrations(db); err != nil {
		return nil, err
	}
	log.Println("Migrations ran")

	return db, nil
}

func executeMigrations(db *sql.DB) error {
	migrations := [...]string{
		`
		alter table videos
		add column nickname text;

		alter table videos
		add column tags text;
		`,
		`
		create table if not exists video_update (
			id integer primary key,
			last_update text
		);

		insert into video_update (id, last_update) values (1, null)
		on conflict (id) do nothing;
		`,
	}

	rows, err := db.Query("select version from migrations where id = 1")
	if err != nil {
		return err
	}

	var version int
	if !rows.Next() {
		rows.Close()
		return fmt.Errorf("No data found in migrations table, should not be possible")
	}

	if err = rows.Scan(&version); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	trans, err := db.Begin()
	if err != nil {
		return err
	}

	for _, migration := range migrations[version:] {
		_, err := trans.Exec(migration)
		if err != nil {
			_ = trans.Rollback()
			return err
		}
	}

	_, err = trans.Exec("update migrations set version = :version where id = 1", len(migrations))
	if err != nil {
		_ = trans.Rollback()
		return err
	}

	return trans.Commit()
}

func readVideoFromRow(rows *sql.Rows) (Video, error) {
	var video Video
	var tags sql.NullString
	err := rows.Scan(
		&video.Id,
		&video.Filename,
		&video.Nickname,
		&tags,
		&video.CreatedAt,
		&video.Status,
	)
	if err != nil {
		return Video{}, err
	}

	if tags.Valid && len(tags.String) > 0 {
		video.Tags = strings.Split(tags.String, ",")
	} else {
		video.Tags = []string{}
	}

	return video, nil
}

func readVideoJsonFile(path string) (VideoJsonFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return VideoJsonFile{}, err
	}

	var payload VideoJsonFile
	err = json.Unmarshal(content, &payload)
	if err != nil {
		return VideoJsonFile{}, err
	}

	return payload, nil
}
