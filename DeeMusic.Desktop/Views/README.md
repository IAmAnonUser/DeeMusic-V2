# DeeMusic Desktop Views

This directory contains all XAML views for the DeeMusic Desktop application.

## Implemented Views

### MainWindow.xaml
The main application window with:
- Custom window chrome (borderless with custom title bar)
- Top bar with search box, theme toggle, and settings button
- Collapsible sidebar navigation menu
- Content area for displaying different views
- Window control buttons (minimize, maximize, close)

### SearchView.xaml
Search results display with:
- Tab-based search type selection (Tracks, Albums, Artists, Playlists)
- Grid layout for search results
- Result cards with artwork and hover effects
- Download buttons on each result
- Loading indicator and empty state

### QueueView.xaml
Download queue management with:
- Header with queue statistics
- Stats bar showing downloading, pending, completed, and failed counts
- Virtualized list view for performance with large queues
- Queue item cards with progress bars
- Control buttons (pause, resume, retry, cancel)
- Status indicators and badges
- Pagination controls for large queues
- Empty state display

### SettingsView.xaml
Application settings with categorized sections:
- **Deezer Settings**: ARL token configuration
- **Download Settings**: Output directory, quality, concurrent downloads, artwork, filename template
- **Lyrics Settings**: Embed lyrics, save lyrics files
- **Network Settings**: Timeout, retry attempts, proxy configuration
- **UI Settings**: Theme, language, system integration options
- Save button to persist changes

### AlbumDetailView.xaml
Album details display with:
- Large album artwork with shadow effect
- Album metadata (title, artist, year, track count, duration)
- Download album button
- Track listing with individual download buttons
- Track numbers and durations

### ArtistDetailView.xaml
Artist page with:
- Circular artist picture
- Artist metadata (name, fan count, album count)
- Albums grid with clickable cards
- Navigation to album details
- Hover effects on album cards

### PlaylistDetailView.xaml
Playlist details with:
- Playlist cover artwork
- Playlist metadata (title, creator, description, track count, duration, fans)
- Download playlist button
- Track listing with album thumbnails
- Track info (title, artist, album, duration)
- Individual track download buttons

## Design Features

### Material Design
All views use Material Design components from MaterialDesignThemes:
- Cards with rounded corners
- Icon buttons with ripple effects
- Outlined text boxes and combo boxes
- Progress bars and indicators
- Pack icons for consistent iconography

### Responsive Layout
- Flexible grid and stack panel layouts
- Scroll viewers for content overflow
- Wrap panels for responsive card grids
- Virtual scrolling for performance

### Animations
- Smooth hover effects with opacity transitions
- Storyboard animations for interactive elements
- Theme transition support

### Theme Support
- Dynamic resource bindings for colors
- Support for dark and light themes
- Consistent color palette across all views

## Data Binding

All views use MVVM pattern with data binding to ViewModels:
- Two-way binding for user inputs
- Command bindings for user actions
- Collection bindings for lists and grids
- Converter bindings for visibility and formatting

## Performance Optimizations

- Virtual scrolling in queue view for large lists
- Lazy loading with pagination
- Efficient item templates
- Recycling virtualization mode

## Next Steps

To complete the UI implementation:
1. Implement custom controls (task 9)
2. Implement theme system (task 10)
3. Wire up ViewModels to backend services
4. Test all views with real data
5. Add keyboard shortcuts and accessibility features
