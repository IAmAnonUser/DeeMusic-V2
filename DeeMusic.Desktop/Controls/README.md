# Custom WPF Controls

This directory contains custom WPF controls for the DeeMusic Desktop application.

## Controls

### 1. ModernButton

A modern button control with ripple animation effects and multiple style variants.

**Features:**
- Ripple animation on click
- Three style variants: Primary, Secondary, Icon
- Icon support
- Hover effects
- Command binding support

**Usage:**
```xaml
<controls:ModernButton 
    Text="Download"
    Icon="⬇"
    ButtonStyle="Primary"
    Command="{Binding DownloadCommand}"/>

<controls:ModernButton 
    Text="Cancel"
    ButtonStyle="Secondary"
    Command="{Binding CancelCommand}"/>

<controls:ModernButton 
    Icon="⚙"
    ButtonStyle="Icon"
    Command="{Binding SettingsCommand}"/>
```

**Properties:**
- `Text` (string): Button text
- `Icon` (string): Icon character/emoji
- `ButtonStyle` (ButtonStyleType): Primary, Secondary, or Icon
- `Command` (ICommand): Command to execute on click
- `CommandParameter` (object): Parameter for command

### 2. ProgressCard

A card control for displaying download queue items with animated progress bars and status indicators.

**Features:**
- Animated progress bar
- Status indicators with color coding
- Control buttons (pause, resume, retry, cancel)
- Speed and ETA display
- Automatic button visibility based on status

**Usage:**
```xaml
<controls:ProgressCard 
    Title="{Binding Title}"
    Artist="{Binding Artist}"
    Progress="{Binding Progress}"
    Status="{Binding Status}"
    Speed="{Binding Speed}"
    ETA="{Binding ETA}"
    PauseCommand="{Binding PauseCommand}"
    ResumeCommand="{Binding ResumeCommand}"
    CancelCommand="{Binding CancelCommand}"
    RetryCommand="{Binding RetryCommand}"/>
```

**Properties:**
- `Title` (string): Track title
- `Artist` (string): Artist name
- `Progress` (double): Progress percentage (0-100)
- `Status` (string): Status text (Downloading, Pending, Paused, Completed, Failed, Cancelled)
- `Speed` (string): Download speed
- `ETA` (string): Estimated time remaining
- `PauseCommand` (ICommand): Command to pause download
- `ResumeCommand` (ICommand): Command to resume download
- `CancelCommand` (ICommand): Command to cancel download
- `RetryCommand` (ICommand): Command to retry failed download

**Status Colors:**
- Downloading: Green (#10b981)
- Pending: Orange (#f59e0b)
- Paused: Gray (#6b7280)
- Completed: Green (#10b981)
- Failed: Red (#ef4444)
- Cancelled: Gray (#6b7280)

### 3. SearchResultCard

A card control for displaying search results (tracks, albums, artists, playlists) with artwork and action buttons.

**Features:**
- Artwork display with placeholder
- Hover effects
- Download button
- Info/View button (for albums, artists, playlists)
- Click-through support
- Adaptive layout based on result type

**Usage:**
```xaml
<controls:SearchResultCard 
    Title="{Binding Title}"
    Artist="{Binding Artist}"
    Album="{Binding Album}"
    Duration="{Binding Duration}"
    Year="{Binding Year}"
    ArtworkUrl="{Binding ArtworkUrl}"
    ResultType="Track"
    DownloadCommand="{Binding DownloadCommand}"
    InfoCommand="{Binding InfoCommand}"
    ItemClickCommand="{Binding ItemClickCommand}"/>
```

**Properties:**
- `Title` (string): Title of the item
- `Artist` (string): Artist name
- `Album` (string): Album name (for tracks)
- `Duration` (string): Duration (for tracks)
- `Year` (string): Release year
- `ArtworkUrl` (string): URL to artwork image
- `ResultType` (SearchResultType): Track, Album, Artist, or Playlist
- `DownloadCommand` (ICommand): Command to download item
- `InfoCommand` (ICommand): Command to view details
- `ItemClickCommand` (ICommand): Command when card is clicked

**Result Type Behavior:**
- **Track**: Shows download button, duration, album info
- **Album**: Shows download album button and view button
- **Artist**: Shows only view button (no download)
- **Playlist**: Shows download playlist button and view button

## Styling

All controls use the dark theme color palette defined in the design document:

- Background: #0a0a0a, #1a1a1a
- Surface: #1a1a1a, #242424
- Primary: #6366f1 (Indigo)
- Success: #10b981 (Green)
- Error: #ef4444 (Red)
- Warning: #f59e0b (Orange)
- Text: #ffffff
- Text Secondary: #9ca3af, #6b7280

## Animation

All controls include smooth animations:
- Button ripple: 600ms cubic ease-out
- Progress bar: 300ms cubic ease-out
- Hover transitions: 150-200ms ease

## Integration

To use these controls in your XAML files:

1. Add namespace reference:
```xaml
xmlns:controls="clr-namespace:DeeMusic.Desktop.Controls"
```

2. Use the controls as shown in the usage examples above.

## Requirements

These controls fulfill the following requirements:
- Requirement 4.4: Real-time progress updates with callbacks
- Requirement 4.5: Modern, responsive UI design
