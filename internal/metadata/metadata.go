package metadata

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bogem/id3v2/v2"
	"github.com/go-flac/flacvorbis"
	"github.com/go-flac/go-flac"
)

// Manager handles metadata operations for audio files
type Manager struct {
	config *Config
}

// Config contains metadata configuration
type Config struct {
	EmbedArtwork bool
	ArtworkSize  int
}

// TrackMetadata contains all metadata for a track
type TrackMetadata struct {
	Title       string
	Artist      string
	Album       string
	AlbumArtist string
	TrackNumber int
	DiscNumber  int
	TotalDiscs  int    // Total number of discs in the album
	Year        int
	Genre       string
	Duration    int
	ISRC        string
	Label       string
	Copyright   string
	ArtworkData []byte
	ArtworkMIME string
}

// NewManager creates a new metadata manager
func NewManager(config *Config) *Manager {
	if config == nil {
		config = &Config{
			EmbedArtwork: true,
			ArtworkSize:  1200,
		}
	}
	return &Manager{
		config: config,
	}
}

// ApplyMetadata applies metadata to an audio file (MP3 or FLAC)
func (m *Manager) ApplyMetadata(filePath string, metadata *TrackMetadata) error {
	if metadata == nil {
		return fmt.Errorf("metadata cannot be nil")
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".mp3":
		return m.applyMP3Metadata(filePath, metadata)
	case ".flac":
		return m.applyFLACMetadata(filePath, metadata)
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}
}

// applyMP3Metadata applies metadata to an MP3 file using ID3v2
func (m *Manager) applyMP3Metadata(filePath string, metadata *TrackMetadata) error {
	// Open MP3 file
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open MP3 file: %w", err)
	}
	defer tag.Close()

	// Set ID3v2.4 version
	tag.SetVersion(4)

	// Set basic text frames
	if metadata.Title != "" {
		tag.SetTitle(metadata.Title)
	}
	if metadata.Artist != "" {
		tag.SetArtist(metadata.Artist)
	}
	if metadata.Album != "" {
		tag.SetAlbum(metadata.Album)
	}
	if metadata.Genre != "" {
		tag.SetGenre(metadata.Genre)
	}

	// Set year
	if metadata.Year > 0 {
		tag.SetYear(strconv.Itoa(metadata.Year))
	}

	// Set album artist (TPE2 frame)
	if metadata.AlbumArtist != "" {
		// Try to delete existing frame first
		tag.DeleteFrames("TPE2")
		// Add new frame
		tag.AddTextFrame("TPE2", id3v2.EncodingUTF8, metadata.AlbumArtist)
	}

	// Set track number with disc number if multi-disc
	if metadata.TrackNumber > 0 {
		trackStr := strconv.Itoa(metadata.TrackNumber)
		tag.AddTextFrame(tag.CommonID("Track number/Position in set"), id3v2.EncodingUTF8, trackStr)
	}

	// Set disc number for multi-disc albums (TPOS frame)
	// Format: "disc/total" (e.g., "2/3" for disc 2 of 3)
	if metadata.DiscNumber > 0 {
		var discStr string
		if metadata.TotalDiscs > 0 {
			discStr = fmt.Sprintf("%d/%d", metadata.DiscNumber, metadata.TotalDiscs)
		} else {
			discStr = strconv.Itoa(metadata.DiscNumber)
		}
		// Try to delete existing frame first
		tag.DeleteFrames("TPOS")
		// Add new frame
		tag.AddTextFrame("TPOS", id3v2.EncodingUTF8, discStr)
	}

	// Set ISRC (International Standard Recording Code)
	if metadata.ISRC != "" {
		tag.AddTextFrame(tag.CommonID("ISRC"), id3v2.EncodingUTF8, metadata.ISRC)
	}

	// Set label/publisher
	if metadata.Label != "" {
		tag.AddTextFrame(tag.CommonID("Publisher"), id3v2.EncodingUTF8, metadata.Label)
	}

	// Set copyright
	if metadata.Copyright != "" {
		tag.AddTextFrame(tag.CommonID("Copyright message"), id3v2.EncodingUTF8, metadata.Copyright)
	}

	// Embed artwork if enabled and available
	if m.config.EmbedArtwork && len(metadata.ArtworkData) > 0 {
		pic := id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    metadata.ArtworkMIME,
			PictureType: id3v2.PTFrontCover,
			Description: "Front Cover",
			Picture:     metadata.ArtworkData,
		}
		tag.AddAttachedPicture(pic)
	}

	// Save changes
	if err := tag.Save(); err != nil {
		return fmt.Errorf("failed to save MP3 metadata: %w", err)
	}

	return nil
}

// applyFLACMetadata applies metadata to a FLAC file using Vorbis comments
func (m *Manager) applyFLACMetadata(filePath string, metadata *TrackMetadata) error {
	// Open FLAC file
	f, err := flac.ParseFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse FLAC file: %w", err)
	}

	// Get or create Vorbis comment block
	var cmtBlock *flac.MetaDataBlock
	for _, block := range f.Meta {
		if block.Type == flac.VorbisComment {
			cmtBlock = block
			break
		}
	}

	if cmtBlock == nil {
		// Create new Vorbis comment block
		cmtBlock = &flac.MetaDataBlock{
			Type: flac.VorbisComment,
		}
		f.Meta = append(f.Meta, cmtBlock)
	}

	// Parse existing comments
	cmt, err := flacvorbis.ParseFromMetaDataBlock(*cmtBlock)
	if err != nil {
		// Create new comment if parsing fails
		cmt = flacvorbis.New()
	}

	// Set metadata fields
	if metadata.Title != "" {
		cmt.Add("TITLE", metadata.Title)
	}
	if metadata.Artist != "" {
		cmt.Add("ARTIST", metadata.Artist)
	}
	if metadata.Album != "" {
		cmt.Add("ALBUM", metadata.Album)
	}
	if metadata.AlbumArtist != "" {
		cmt.Add("ALBUMARTIST", metadata.AlbumArtist)
	}
	if metadata.Genre != "" {
		cmt.Add("GENRE", metadata.Genre)
	}
	if metadata.Year > 0 {
		cmt.Add("DATE", strconv.Itoa(metadata.Year))
	}
	if metadata.TrackNumber > 0 {
		cmt.Add("TRACKNUMBER", strconv.Itoa(metadata.TrackNumber))
	}
	if metadata.DiscNumber > 0 {
		if metadata.TotalDiscs > 0 {
			cmt.Add("DISCNUMBER", fmt.Sprintf("%d/%d", metadata.DiscNumber, metadata.TotalDiscs))
		} else {
			cmt.Add("DISCNUMBER", strconv.Itoa(metadata.DiscNumber))
		}
	}
	if metadata.TotalDiscs > 0 {
		cmt.Add("TOTALDISCS", strconv.Itoa(metadata.TotalDiscs))
	}
	if metadata.ISRC != "" {
		cmt.Add("ISRC", metadata.ISRC)
	}
	if metadata.Label != "" {
		cmt.Add("LABEL", metadata.Label)
	}
	if metadata.Copyright != "" {
		cmt.Add("COPYRIGHT", metadata.Copyright)
	}

	// Marshal comments back to block
	res := cmt.Marshal()
	cmtBlock.Data = res.Data

	// Handle artwork for FLAC
	if m.config.EmbedArtwork && len(metadata.ArtworkData) > 0 {
		// Check if picture block already exists
		hasPicture := false
		for _, block := range f.Meta {
			if block.Type == flac.Picture {
				hasPicture = true
				break
			}
		}

		// Add picture block if not present
		if !hasPicture {
			picBlock := &flac.MetaDataBlock{
				Type: flac.Picture,
				Data: m.createFLACPictureBlock(metadata.ArtworkData, metadata.ArtworkMIME),
			}
			f.Meta = append(f.Meta, picBlock)
		}
	}

	// Write back to file
	if err := f.Save(filePath); err != nil {
		return fmt.Errorf("failed to save FLAC file: %w", err)
	}

	return nil
}

// createFLACPictureBlock creates a FLAC picture block from image data
func (m *Manager) createFLACPictureBlock(imageData []byte, mimeType string) []byte {
	// FLAC picture block format:
	// 4 bytes: picture type (3 = front cover)
	// 4 bytes: MIME type length
	// n bytes: MIME type string
	// 4 bytes: description length
	// n bytes: description string
	// 4 bytes: width
	// 4 bytes: height
	// 4 bytes: color depth
	// 4 bytes: number of colors (0 for non-indexed)
	// 4 bytes: picture data length
	// n bytes: picture data

	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	description := "Front Cover"
	
	// Calculate total size
	size := 4 + 4 + len(mimeType) + 4 + len(description) + 4 + 4 + 4 + 4 + 4 + len(imageData)
	data := make([]byte, size)
	
	pos := 0
	
	// Picture type (3 = front cover)
	writeUint32BE(data[pos:], 3)
	pos += 4
	
	// MIME type length and string
	writeUint32BE(data[pos:], uint32(len(mimeType)))
	pos += 4
	copy(data[pos:], mimeType)
	pos += len(mimeType)
	
	// Description length and string
	writeUint32BE(data[pos:], uint32(len(description)))
	pos += 4
	copy(data[pos:], description)
	pos += len(description)
	
	// Width, height, color depth, colors (all 0 - will be determined by decoder)
	writeUint32BE(data[pos:], 0)
	pos += 4
	writeUint32BE(data[pos:], 0)
	pos += 4
	writeUint32BE(data[pos:], 0)
	pos += 4
	writeUint32BE(data[pos:], 0)
	pos += 4
	
	// Picture data length and data
	writeUint32BE(data[pos:], uint32(len(imageData)))
	pos += 4
	copy(data[pos:], imageData)
	
	return data
}

// writeUint32BE writes a uint32 in big-endian format
func writeUint32BE(b []byte, v uint32) {
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
}

// RemoveMetadata removes all metadata from an audio file
func (m *Manager) RemoveMetadata(filePath string) error {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".mp3":
		return m.removeMP3Metadata(filePath)
	case ".flac":
		return m.removeFLACMetadata(filePath)
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}
}

// removeMP3Metadata removes all ID3 tags from an MP3 file
func (m *Manager) removeMP3Metadata(filePath string) error {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open MP3 file: %w", err)
	}
	defer tag.Close()

	tag.DeleteAllFrames()
	
	if err := tag.Save(); err != nil {
		return fmt.Errorf("failed to save MP3 file: %w", err)
	}

	return nil
}

// removeFLACMetadata removes all Vorbis comments from a FLAC file
func (m *Manager) removeFLACMetadata(filePath string) error {
	f, err := flac.ParseFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse FLAC file: %w", err)
	}

	// Remove Vorbis comment and picture blocks
	newMeta := make([]*flac.MetaDataBlock, 0)
	for _, block := range f.Meta {
		if block.Type != flac.VorbisComment && block.Type != flac.Picture {
			newMeta = append(newMeta, block)
		}
	}
	f.Meta = newMeta

	if err := f.Save(filePath); err != nil {
		return fmt.Errorf("failed to save FLAC file: %w", err)
	}

	return nil
}

// GetMetadata reads metadata from an audio file
func (m *Manager) GetMetadata(filePath string) (*TrackMetadata, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".mp3":
		return m.getMP3Metadata(filePath)
	case ".flac":
		return m.getFLACMetadata(filePath)
	default:
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}
}

// getMP3Metadata reads metadata from an MP3 file
func (m *Manager) getMP3Metadata(filePath string) (*TrackMetadata, error) {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return nil, fmt.Errorf("failed to open MP3 file: %w", err)
	}
	defer tag.Close()

	metadata := &TrackMetadata{
		Title:  tag.Title(),
		Artist: tag.Artist(),
		Album:  tag.Album(),
		Genre:  tag.Genre(),
	}

	// Parse year
	if yearStr := tag.Year(); yearStr != "" {
		if year, err := strconv.Atoi(yearStr); err == nil {
			metadata.Year = year
		}
	}

	// Get album artist
	if frames := tag.GetFrames(tag.CommonID("Band/Orchestra/Accompaniment")); len(frames) > 0 {
		if tf, ok := frames[0].(id3v2.TextFrame); ok {
			metadata.AlbumArtist = tf.Text
		}
	}

	// Get track number
	if frames := tag.GetFrames(tag.CommonID("Track number/Position in set")); len(frames) > 0 {
		if tf, ok := frames[0].(id3v2.TextFrame); ok {
			if trackNum, err := strconv.Atoi(strings.Split(tf.Text, "/")[0]); err == nil {
				metadata.TrackNumber = trackNum
			}
		}
	}

	// Get disc number
	if frames := tag.GetFrames(tag.CommonID("Part of a set")); len(frames) > 0 {
		if tf, ok := frames[0].(id3v2.TextFrame); ok {
			if discNum, err := strconv.Atoi(strings.Split(tf.Text, "/")[0]); err == nil {
				metadata.DiscNumber = discNum
			}
		}
	}

	return metadata, nil
}

// getFLACMetadata reads metadata from a FLAC file
func (m *Manager) getFLACMetadata(filePath string) (*TrackMetadata, error) {
	f, err := flac.ParseFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse FLAC file: %w", err)
	}

	metadata := &TrackMetadata{}

	// Find Vorbis comment block
	for _, block := range f.Meta {
		if block.Type == flac.VorbisComment {
			cmt, err := flacvorbis.ParseFromMetaDataBlock(*block)
			if err != nil {
				continue
			}

			// Extract fields
			if titles, err := cmt.Get("TITLE"); err == nil && len(titles) > 0 {
				metadata.Title = titles[0]
			}
			if artists, err := cmt.Get("ARTIST"); err == nil && len(artists) > 0 {
				metadata.Artist = artists[0]
			}
			if albums, err := cmt.Get("ALBUM"); err == nil && len(albums) > 0 {
				metadata.Album = albums[0]
			}
			if albumArtists, err := cmt.Get("ALBUMARTIST"); err == nil && len(albumArtists) > 0 {
				metadata.AlbumArtist = albumArtists[0]
			}
			if genres, err := cmt.Get("GENRE"); err == nil && len(genres) > 0 {
				metadata.Genre = genres[0]
			}
			if dates, err := cmt.Get("DATE"); err == nil && len(dates) > 0 {
				if year, err := strconv.Atoi(dates[0]); err == nil {
					metadata.Year = year
				}
			}
			if trackNums, err := cmt.Get("TRACKNUMBER"); err == nil && len(trackNums) > 0 {
				if trackNum, err := strconv.Atoi(trackNums[0]); err == nil {
					metadata.TrackNumber = trackNum
				}
			}
			if discNums, err := cmt.Get("DISCNUMBER"); err == nil && len(discNums) > 0 {
				if discNum, err := strconv.Atoi(discNums[0]); err == nil {
					metadata.DiscNumber = discNum
				}
			}

			break
		}
	}

	return metadata, nil
}

// FileExists checks if a file exists
func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}
