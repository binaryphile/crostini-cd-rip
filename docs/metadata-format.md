# Manual Metadata Format

When MusicBrainz doesn't have your disc, provide metadata as JSON.

## JSON Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| artist | string | yes | Album artist (use "Various Artists" for compilations) |
| album | string | yes | Album title |
| year | string | no | Release year (YYYY) |
| genre | string | no | Genre |
| disc | int | no | Disc number (for multi-disc sets) |
| totalDiscs | int | no | Total discs in set |
| totalTracks | int | no | Total tracks on disc |
| coverArt | string | no | Path to cover image file |
| tracks | array | yes | Track listing |
| tracks[].num | int | yes | Track number |
| tracks[].title | string | yes | Track title |
| tracks[].artist | string | no | Track artist (required for compilations) |

## Example: Compilation Album

```json
{
  "artist": "Various Artists",
  "album": "Ultra Mix - Dance Hits Of The 80's And 90's",
  "year": "1999",
  "genre": "Dance",
  "disc": 2,
  "totalDiscs": 2,
  "tracks": [
    {"num": 1, "artist": "C + C Music Factory", "title": "Gonna Make You Sweat (Everybody Dance Now)"},
    {"num": 2, "artist": "EMF", "title": "Unbelievable"},
    {"num": 3, "artist": "Technotronic", "title": "Pump Up The Jam"}
  ]
}
```

## Example: Standard Album

```json
{
  "artist": "Pink Floyd",
  "album": "The Dark Side of the Moon",
  "year": "1973",
  "genre": "Progressive Rock",
  "coverArt": "/tmp/cover.jpg",
  "tracks": [
    {"num": 1, "title": "Speak to Me"},
    {"num": 2, "title": "Breathe"},
    {"num": 3, "title": "On the Run"},
    {"num": 4, "title": "Time"}
  ]
}
```

## Example: Multi-Disc Set (Disc 1)

```json
{
  "artist": "The Beatles",
  "album": "The White Album",
  "year": "1968",
  "genre": "Rock",
  "disc": 1,
  "totalDiscs": 2,
  "totalTracks": 17,
  "tracks": [
    {"num": 1, "title": "Back in the U.S.S.R."},
    {"num": 2, "title": "Dear Prudence"},
    {"num": 3, "title": "Glass Onion"}
  ]
}
```

Rip and encode each disc separately. The `disc` field ensures filenames include `CD1`, `CD2`, etc.

## For Claude: Extracting Metadata

This workflow is format-agnostic. Users may provide metadata from any source:
- Discogs page (markdown or HTML download)
- AllMusic or Wikipedia
- CD liner notes (typed or photographed)
- Amazon or other retailer listing

### Extraction Steps

1. Find album title (usually in heading or product name)
2. Find artist - look for "Various" or "Various Artists" for compilations
3. Find year in release info
4. Extract tracklist:
   - Position → track number (handle "1-1", "A1", "1.01" formats for multi-disc)
   - Artist → track artist (for compilations)
   - Title → track title
5. Normalize characters (see below)
6. Output as JSON matching schema above
7. Save to /tmp/metadata.json
8. Pre-flight check: `ls /tmp/cd-rip/*.wav | wc -l` should match track count
9. Run: `cd-encode --metadata /tmp/metadata.json /tmp/cd-rip`

### Character Normalization

Normalize text when extracting to avoid encoding issues in filenames and tags:

| Source Character | Replace With |
|------------------|--------------|
| Smart quotes `"` `"` `'` `'` | Straight quotes `"` `'` |
| En/em dashes `–` `—` | Hyphen `-` |
| Ellipsis `…` | Three periods `...` |
| Accented letters `é` `ö` `ñ` | Keep as-is (encoder handles) |
| Non-breaking space | Regular space |
| Fancy apostrophes `'` | Straight apostrophe `'` |

The encoder's `sanitize()` function handles most edge cases, but clean input prevents surprises.

### Troubleshooting

| Problem | Solution |
|---------|----------|
| Track count mismatch | Verify `totalTracks` matches WAV file count in /tmp/cd-rip |
| Wrong disc | Check `disc` field matches the physical disc ripped |
| Encoding fails | Ensure track numbers are sequential starting at 1 |
| Missing track artist | Required for compilations (`artist: "Various Artists"`) |
