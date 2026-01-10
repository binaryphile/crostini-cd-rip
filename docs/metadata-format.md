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

## For Claude: Extracting from Discogs

When user provides a Discogs page (markdown or HTML):

1. Find album title (usually in heading)
2. Find artist - look for "Various" or specific artist name
3. Find year in release info
4. Extract tracklist table:
   - Position column → track number (handle "1-1" format for multi-disc)
   - Artist column → track artist
   - Title column → track title
5. Output as JSON matching schema above
6. Save to /tmp/metadata.json
7. Run: `cd-encode --metadata /tmp/metadata.json /tmp/cd-rip`
