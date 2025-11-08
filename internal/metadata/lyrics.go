package metadata

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bogem/id3v2/v2"
	"github.com/go-flac/flacvorbis"
	"github.com/go-flac/go-flac"
)

// LyricsConfig contains lyrics-related configuration
type LyricsConfig struct {
	EmbedInFile      bool
	SaveSeparateFile bool
	Language         string
}

// Lyrics represents track lyrics
type Lyrics struct {
	SyncedLyrics   string
	UnsyncedLyrics string
	Language       string
}

// EmbedLyrics embeds lyrics in an audio file
func (m *Manager) EmbedLyrics(filePath string, lyrics *Lyrics, config *LyricsConfig) error {
	if lyrics == nil {
		return fmt.Errorf("lyrics cannot be nil")
	}

	if config == nil {
		config = &LyricsConfig{
			EmbedInFile:      true,
			SaveSeparateFile: false,
			Language:         "eng",
		}
	}

	// Embed in file if enabled
	if config.EmbedInFile {
		ext := strings.ToLower(filepath.Ext(filePath))
		switch ext {
		case ".mp3":
			if err := m.embedMP3Lyrics(filePath, lyrics, config); err != nil {
				return fmt.Errorf("failed to embed MP3 lyrics: %w", err)
			}
		case ".flac":
			if err := m.embedFLACLyrics(filePath, lyrics); err != nil {
				return fmt.Errorf("failed to embed FLAC lyrics: %w", err)
			}
		default:
			return fmt.Errorf("unsupported file format for lyrics: %s", ext)
		}
	}

	// Save separate files if enabled
	if config.SaveSeparateFile {
		if err := m.saveLyricsFiles(filePath, lyrics); err != nil {
			return fmt.Errorf("failed to save lyrics files: %w", err)
		}
	}

	return nil
}

// embedMP3Lyrics embeds lyrics in an MP3 file using ID3v2 tags
func (m *Manager) embedMP3Lyrics(filePath string, lyrics *Lyrics, config *LyricsConfig) error {
	// Open MP3 file
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open MP3 file: %w", err)
	}
	defer tag.Close()

	// Remove existing lyrics frames
	tag.DeleteFrames(tag.CommonID("Unsynchronised lyrics/text transcription"))
	tag.DeleteFrames(tag.CommonID("Synchronised lyrics/text"))

	// Add unsynchronized lyrics (USLT frame)
	if lyrics.UnsyncedLyrics != "" {
		usltFrame := id3v2.UnsynchronisedLyricsFrame{
			Encoding:          id3v2.EncodingUTF8,
			Language:          config.Language,
			ContentDescriptor: "",
			Lyrics:            lyrics.UnsyncedLyrics,
		}
		tag.AddUnsynchronisedLyricsFrame(usltFrame)
	}

	// Add synchronized lyrics (SYLT frame) if available
	if lyrics.SyncedLyrics != "" {
		// Parse LRC format to create SYLT frame
		syltFrame := m.createSYLTFrame(lyrics.SyncedLyrics, config.Language)
		if syltFrame != nil {
			tag.AddFrame(tag.CommonID("Synchronised lyrics/text"), syltFrame)
		}
	}

	// Save changes
	if err := tag.Save(); err != nil {
		return fmt.Errorf("failed to save MP3 file: %w", err)
	}

	return nil
}

// createSYLTFrame creates a SYLT (Synchronized Lyrics) frame from LRC format
func (m *Manager) createSYLTFrame(lrcLyrics string, language string) id3v2.Framer {
	// Parse LRC format
	lines := strings.Split(lrcLyrics, "\n")
	
	// Build SYLT frame data
	// SYLT frame format:
	// - Text encoding (1 byte)
	// - Language (3 bytes)
	// - Time stamp format (1 byte): 0x02 = milliseconds
	// - Content type (1 byte): 0x01 = lyrics
	// - Content descriptor (null-terminated string)
	// - Synchronized text (text + timestamp pairs)
	
	var frameData []byte
	
	// Text encoding (UTF-8)
	frameData = append(frameData, id3v2.EncodingUTF8.Key)
	
	// Language (3 bytes)
	if len(language) < 3 {
		language = "eng"
	}
	frameData = append(frameData, []byte(language[:3])...)
	
	// Time stamp format (milliseconds)
	frameData = append(frameData, 0x02)
	
	// Content type (lyrics)
	frameData = append(frameData, 0x01)
	
	// Content descriptor (empty, null-terminated)
	frameData = append(frameData, 0x00)
	
	// Parse and add synchronized text
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "[") {
			continue
		}
		
		// Parse LRC line: [mm:ss.xx]text
		closeBracket := strings.Index(line, "]")
		if closeBracket == -1 {
			continue
		}
		
		timestamp := line[1:closeBracket]
		text := strings.TrimSpace(line[closeBracket+1:])
		
		// Convert timestamp to milliseconds
		ms := m.lrcTimestampToMilliseconds(timestamp)
		if ms < 0 {
			continue
		}
		
		// Add text (null-terminated)
		frameData = append(frameData, []byte(text)...)
		frameData = append(frameData, 0x00)
		
		// Add timestamp (4 bytes, big-endian)
		frameData = append(frameData, byte(ms>>24))
		frameData = append(frameData, byte(ms>>16))
		frameData = append(frameData, byte(ms>>8))
		frameData = append(frameData, byte(ms))
	}
	
	// Create custom frame
	return id3v2.UnknownFrame{Body: frameData}
}

// lrcTimestampToMilliseconds converts LRC timestamp to milliseconds
func (m *Manager) lrcTimestampToMilliseconds(timestamp string) int {
	// Format: mm:ss.xx or mm:ss.xxx
	parts := strings.Split(timestamp, ":")
	if len(parts) != 2 {
		return -1
	}
	
	var minutes int
	n, err := fmt.Sscanf(parts[0], "%d", &minutes)
	if err != nil || n != 1 {
		return -1
	}
	
	secondsParts := strings.Split(parts[1], ".")
	if len(secondsParts) != 2 {
		return -1
	}
	
	var seconds int
	n, err = fmt.Sscanf(secondsParts[0], "%d", &seconds)
	if err != nil || n != 1 {
		return -1
	}
	
	// Handle both .xx and .xxx formats
	centiseconds := secondsParts[1]
	if len(centiseconds) == 2 {
		centiseconds += "0"
	}
	var ms int
	n, err = fmt.Sscanf(centiseconds, "%d", &ms)
	if err != nil || n != 1 {
		return -1
	}
	
	totalMs := (minutes*60+seconds)*1000 + ms
	return totalMs
}

// embedFLACLyrics embeds lyrics in a FLAC file using Vorbis comments
func (m *Manager) embedFLACLyrics(filePath string, lyrics *Lyrics) error {
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

	// Add lyrics to Vorbis comments
	// FLAC uses LYRICS field for unsynced lyrics
	if lyrics.UnsyncedLyrics != "" {
		cmt.Add("LYRICS", lyrics.UnsyncedLyrics)
	}

	// For synchronized lyrics, we can store them in a custom field
	if lyrics.SyncedLyrics != "" {
		cmt.Add("SYNCEDLYRICS", lyrics.SyncedLyrics)
	}

	// Marshal comments back to block
	res := cmt.Marshal()
	cmtBlock.Data = res.Data

	// Write back to file
	if err := f.Save(filePath); err != nil {
		return fmt.Errorf("failed to save FLAC file: %w", err)
	}

	return nil
}

// saveLyricsFiles saves lyrics as separate .lrc and .txt files
func (m *Manager) saveLyricsFiles(audioFilePath string, lyrics *Lyrics) error {
	// Get base path without extension
	basePath := strings.TrimSuffix(audioFilePath, filepath.Ext(audioFilePath))

	// Save synchronized lyrics as .lrc file
	if lyrics.SyncedLyrics != "" {
		lrcPath := basePath + ".lrc"
		if err := os.WriteFile(lrcPath, []byte(lyrics.SyncedLyrics), 0644); err != nil {
			return fmt.Errorf("failed to save LRC file: %w", err)
		}
	}

	// Save plain text lyrics as .txt file
	if lyrics.UnsyncedLyrics != "" {
		txtPath := basePath + ".txt"
		if err := os.WriteFile(txtPath, []byte(lyrics.UnsyncedLyrics), 0644); err != nil {
			return fmt.Errorf("failed to save TXT file: %w", err)
		}
	}

	return nil
}

// GetLyrics reads lyrics from an audio file
func (m *Manager) GetLyrics(filePath string) (*Lyrics, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".mp3":
		return m.getMP3Lyrics(filePath)
	case ".flac":
		return m.getFLACLyrics(filePath)
	default:
		return nil, fmt.Errorf("unsupported file format for lyrics: %s", ext)
	}
}

// getMP3Lyrics reads lyrics from an MP3 file
func (m *Manager) getMP3Lyrics(filePath string) (*Lyrics, error) {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return nil, fmt.Errorf("failed to open MP3 file: %w", err)
	}
	defer tag.Close()

	lyrics := &Lyrics{}

	// Get unsynchronized lyrics (USLT frame)
	usltFrames := tag.GetFrames(tag.CommonID("Unsynchronised lyrics/text transcription"))
	if len(usltFrames) > 0 {
		if uslt, ok := usltFrames[0].(id3v2.UnsynchronisedLyricsFrame); ok {
			lyrics.UnsyncedLyrics = uslt.Lyrics
			lyrics.Language = uslt.Language
		}
	}

	// Get synchronized lyrics (SYLT frame)
	syltFrames := tag.GetFrames(tag.CommonID("Synchronised lyrics/text"))
	if len(syltFrames) > 0 {
		// Parse SYLT frame to LRC format
		if sylt, ok := syltFrames[0].(id3v2.UnknownFrame); ok {
			lyrics.SyncedLyrics = m.parseSYLTFrame(sylt.Body)
		}
	}

	return lyrics, nil
}

// parseSYLTFrame parses a SYLT frame to LRC format
func (m *Manager) parseSYLTFrame(frameData []byte) string {
	if len(frameData) < 6 {
		return ""
	}

	// Skip encoding (1), language (3), timestamp format (1), content type (1)
	pos := 6

	// Skip content descriptor (null-terminated string)
	for pos < len(frameData) && frameData[pos] != 0 {
		pos++
	}
	pos++ // Skip null terminator

	// Parse synchronized text
	var lrcBuilder strings.Builder
	for pos < len(frameData) {
		// Read text until null terminator
		textStart := pos
		for pos < len(frameData) && frameData[pos] != 0 {
			pos++
		}
		text := string(frameData[textStart:pos])
		pos++ // Skip null terminator

		// Read timestamp (4 bytes)
		if pos+4 > len(frameData) {
			break
		}
		ms := int(frameData[pos])<<24 | int(frameData[pos+1])<<16 | int(frameData[pos+2])<<8 | int(frameData[pos+3])
		pos += 4

		// Convert to LRC format
		timestamp := m.millisecondsToLRCTimestamp(ms)
		lrcBuilder.WriteString(fmt.Sprintf("[%s]%s\n", timestamp, text))
	}

	return lrcBuilder.String()
}

// millisecondsToLRCTimestamp converts milliseconds to LRC timestamp format
func (m *Manager) millisecondsToLRCTimestamp(ms int) string {
	if ms < 0 {
		ms = 0
	}

	totalSeconds := ms / 1000
	milliseconds := (ms % 1000) / 10

	minutes := totalSeconds / 60
	seconds := totalSeconds % 60

	return fmt.Sprintf("%02d:%02d.%02d", minutes, seconds, milliseconds)
}

// getFLACLyrics reads lyrics from a FLAC file
func (m *Manager) getFLACLyrics(filePath string) (*Lyrics, error) {
	f, err := flac.ParseFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse FLAC file: %w", err)
	}

	lyrics := &Lyrics{}

	// Find Vorbis comment block
	for _, block := range f.Meta {
		if block.Type == flac.VorbisComment {
			cmt, err := flacvorbis.ParseFromMetaDataBlock(*block)
			if err != nil {
				continue
			}

			// Get unsynced lyrics
			if lyricsFields, err := cmt.Get("LYRICS"); err == nil && len(lyricsFields) > 0 {
				lyrics.UnsyncedLyrics = lyricsFields[0]
			}

			// Get synced lyrics from custom field
			if syncedFields, err := cmt.Get("SYNCEDLYRICS"); err == nil && len(syncedFields) > 0 {
				lyrics.SyncedLyrics = syncedFields[0]
			}

			break
		}
	}

	return lyrics, nil
}

// RemoveLyrics removes all lyrics from an audio file
func (m *Manager) RemoveLyrics(filePath string) error {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".mp3":
		return m.removeMP3Lyrics(filePath)
	case ".flac":
		return m.removeFLACLyrics(filePath)
	default:
		return fmt.Errorf("unsupported file format for lyrics: %s", ext)
	}
}

// removeMP3Lyrics removes lyrics from an MP3 file
func (m *Manager) removeMP3Lyrics(filePath string) error {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open MP3 file: %w", err)
	}
	defer tag.Close()

	// Remove lyrics frames
	tag.DeleteFrames(tag.CommonID("Unsynchronised lyrics/text transcription"))
	tag.DeleteFrames(tag.CommonID("Synchronised lyrics/text"))

	if err := tag.Save(); err != nil {
		return fmt.Errorf("failed to save MP3 file: %w", err)
	}

	return nil
}

// removeFLACLyrics removes lyrics from a FLAC file
func (m *Manager) removeFLACLyrics(filePath string) error {
	f, err := flac.ParseFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse FLAC file: %w", err)
	}

	// Find and update Vorbis comment block
	for _, block := range f.Meta {
		if block.Type == flac.VorbisComment {
			cmt, err := flacvorbis.ParseFromMetaDataBlock(*block)
			if err != nil {
				continue
			}

			// Remove lyrics fields by creating new comment without them
			newCmt := flacvorbis.New()
			newCmt.Vendor = cmt.Vendor
			
			// Copy all comments except lyrics
			for _, comment := range cmt.Comments {
				if !strings.HasPrefix(comment, "LYRICS=") && !strings.HasPrefix(comment, "SYNCEDLYRICS=") {
					newCmt.Comments = append(newCmt.Comments, comment)
				}
			}

			// Marshal back
			res := newCmt.Marshal()
			block.Data = res.Data

			break
		}
	}

	if err := f.Save(filePath); err != nil {
		return fmt.Errorf("failed to save FLAC file: %w", err)
	}

	return nil
}

// LoadLyricsFromFiles loads lyrics from separate .lrc and .txt files
func (m *Manager) LoadLyricsFromFiles(audioFilePath string) (*Lyrics, error) {
	basePath := strings.TrimSuffix(audioFilePath, filepath.Ext(audioFilePath))
	lyrics := &Lyrics{}

	// Load .lrc file if exists
	lrcPath := basePath + ".lrc"
	if data, err := os.ReadFile(lrcPath); err == nil {
		lyrics.SyncedLyrics = string(data)
	}

	// Load .txt file if exists
	txtPath := basePath + ".txt"
	if data, err := os.ReadFile(txtPath); err == nil {
		lyrics.UnsyncedLyrics = string(data)
	}

	// Return error if no lyrics found
	if lyrics.SyncedLyrics == "" && lyrics.UnsyncedLyrics == "" {
		return nil, fmt.Errorf("no lyrics files found")
	}

	return lyrics, nil
}
