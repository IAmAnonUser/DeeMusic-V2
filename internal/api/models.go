package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Track represents a Deezer track
type Track struct {
	ID              FlexibleID `json:"id"`
	Title           string    `json:"title"`
	TitleShort      string    `json:"title_short"`
	TitleVersion    string    `json:"title_version"`
	ISRC            string    `json:"isrc"`
	Link            string    `json:"link"`
	Duration        int       `json:"duration"`
	TrackPosition   int       `json:"track_position"` // Some endpoints use this
	TrackNumber     int       `json:"track_number"`   // Some endpoints use this
	DiscNumber      int       `json:"disk_number"`
	Rank            int       `json:"rank"`
	ExplicitLyrics  bool      `json:"explicit_lyrics"`
	ExplicitContent int       `json:"explicit_content_lyrics"`
	PreviewURL      string    `json:"preview"`
	MD5Image        string    `json:"md5_image"`
	Artist          *Artist   `json:"artist"`
	Album           *Album    `json:"album"`
	Type            string    `json:"type"`
	ReleaseDate     string    `json:"release_date"`
	Available       bool      `json:"readable"`
	Contributors    []*Artist `json:"contributors"`
	
	// Internal fields (not serialized)
	IsMultiDiscAlbum bool      `json:"-"` // Used for folder structure decisions
	TotalDiscs       int       `json:"-"` // Total number of discs in the album
	AlbumArtist      string    `json:"-"` // Album artist (Various Artists for compilations/soundtracks)
	Playlist         *Playlist `json:"-"` // Playlist this track belongs to (for playlist downloads)
	PlaylistPosition int       `json:"-"` // Position in playlist (for playlist downloads)
}

// GetTrackNumber returns the track number, preferring track_number over track_position
func (t *Track) GetTrackNumber() int {
	if t.TrackNumber > 0 {
		return t.TrackNumber
	}
	return t.TrackPosition
}

// Album represents a Deezer album
type Album struct {
	ID              FlexibleID `json:"id"`
	Title           string    `json:"title"`
	UPC             string    `json:"upc"`
	Link            string    `json:"link"`
	Cover           string    `json:"cover"`
	CoverSmall      string    `json:"cover_small"`
	CoverMedium     string    `json:"cover_medium"`
	CoverBig        string    `json:"cover_big"`
	CoverXL         string    `json:"cover_xl"`
	MD5Image        string    `json:"md5_image"`
	GenreID         int       `json:"genre_id"`
	Genres          *Genres   `json:"genres"`
	Label           string    `json:"label"`
	TrackCount      int       `json:"nb_tracks"`
	DiscCount       int       `json:"nb_disk"` // Total number of discs in the album
	Duration        int       `json:"duration"`
	Fans            int       `json:"fans"`
	ReleaseDate     string    `json:"release_date"`
	RecordType      string    `json:"record_type"`
	Available       bool      `json:"available"`
	ExplicitLyrics  bool      `json:"explicit_lyrics"`
	ExplicitContent int       `json:"explicit_content_lyrics"`
	Contributors    []*Artist `json:"contributors"`
	Artist          *Artist   `json:"artist"`
	Type            string    `json:"type"`
	Tracks          *Tracks   `json:"tracks"`
}

// Artist represents a Deezer artist
type Artist struct {
	ID            FlexibleID `json:"id"`
	Name          string     `json:"name"`
	Link          string     `json:"link"`
	Picture       string     `json:"picture"`
	PictureSmall  string     `json:"picture_small"`
	PictureMedium string     `json:"picture_medium"`
	PictureBig    string     `json:"picture_big"`
	PictureXL     string     `json:"picture_xl"`
	TrackList     string     `json:"tracklist"`
	Type          string     `json:"type"`
	Role          string     `json:"role"`
}

// Playlist represents a Deezer playlist
type Playlist struct {
	ID                    FlexibleID `json:"id"`
	Title                 string    `json:"title"`
	Description           string    `json:"description"`
	Duration              int       `json:"duration"`
	Public                bool      `json:"public"`
	IsLovedTrack          bool      `json:"is_loved_track"`
	Collaborative         bool      `json:"collaborative"`
	TrackCount            int       `json:"nb_tracks"`
	Fans                  int       `json:"fans"`
	Link                  string    `json:"link"`
	Picture               string    `json:"picture"`
	PictureSmall          string    `json:"picture_small"`
	PictureMedium         string    `json:"picture_medium"`
	PictureBig            string    `json:"picture_big"`
	PictureXL             string    `json:"picture_xl"`
	Checksum              string    `json:"checksum"`
	Creator               *User     `json:"creator"`
	Type                  string    `json:"type"`
	Tracks                *Tracks   `json:"tracks"`
	CreationDate          FlexibleTime `json:"creation_date"`
	ExplicitContentLyrics int       `json:"explicit_content_lyrics"`
	ExplicitContentCover  int       `json:"explicit_content_cover"`
}

// User represents a Deezer user
type User struct {
	ID        FlexibleID `json:"id"`
	Name      string `json:"name"`
	Link      string `json:"link"`
	Picture   string `json:"picture"`
	Type      string `json:"type"`
	TrackList string `json:"tracklist"`
}

// Tracks represents a collection of tracks with pagination support
type Tracks struct {
	Data  []*Track `json:"data"`
	Total int      `json:"total"`
	Next  string   `json:"next"`
}

// Genres represents a collection of genres
type Genres struct {
	Data []*Genre `json:"data"`
}

// Genre represents a music genre
type Genre struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
	Type    string `json:"type"`
}

// SearchResult represents search results
type SearchResult struct {
	Data  []interface{} `json:"data"`
	Total int           `json:"total"`
	Next  string        `json:"next"`
}

// Lyrics represents track lyrics
type Lyrics struct {
	ID             string           `json:"id"`
	TrackID        string           `json:"track_id"`
	SyncedLyrics   string           `json:"synced_lyrics"`
	UnsyncedLyrics string           `json:"unsynced_lyrics"`
	Synchronized   []*LyricLine     `json:"synchronized"`
	Writers        string           `json:"writers"`
	Copyright      string           `json:"copyright"`
}

// LyricLine represents a single line of synchronized lyrics
type LyricLine struct {
	Line         string  `json:"line"`
	Milliseconds int     `json:"milliseconds"`
	Duration     int     `json:"duration"`
	LrcTimestamp string  `json:"lrc_timestamp"`
}

// DownloadURL represents a track download URL
type DownloadURL struct {
	TrackID  string
	Quality  string
	URL      string
	FileSize int64
	Format   string
}

// Quality constants
const (
	QualityMP3128  = "MP3_128"
	QualityMP3320  = "MP3_320"
	QualityFLAC    = "FLAC"
)

// FlexibleID is a type that can unmarshal from both string and number JSON values
type FlexibleID string

// UnmarshalJSON implements custom unmarshaling for FlexibleID
func (f *FlexibleID) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FlexibleID(s)
		return nil
	}
	
	// Try to unmarshal as number
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*f = FlexibleID(n.String())
		return nil
	}
	
	return fmt.Errorf("FlexibleID must be a string or number")
}

// String returns the string representation
func (f FlexibleID) String() string {
	return string(f)
}

// Int64 returns the int64 representation
func (f FlexibleID) Int64() (int64, error) {
	return strconv.ParseInt(string(f), 10, 64)
}

// FlexibleTime is a type that can unmarshal from multiple time formats
type FlexibleTime struct {
	time.Time
}

// UnmarshalJSON implements custom unmarshaling for FlexibleTime
func (ft *FlexibleTime) UnmarshalJSON(data []byte) error {
	// Remove quotes
	s := string(data)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	
	// Try different time formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			ft.Time = t
			return nil
		}
	}
	
	return fmt.Errorf("unable to parse time: %s", s)
}
