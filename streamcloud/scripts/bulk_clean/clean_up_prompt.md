Act as a precise music archivist. Below is a JSON array of music file objects, each with a "path" (absolute path) and "filename" (basename).

Your task is to identify the correct metadata for EVERY track in the array and return a single JSON array to stdout.

Processing Rules for each track:

0. **Stay Focused**: Use web search only if the metadata is not obvious or needs verification.
1. **Identify Metadata**: Find the true primary Artist, original Studio Album (ignore compilations, "Greatest Hits", or playlists), and correct Track Name.
2. **Collaborations**: For collaborations (e.g., "Artist A & Artist B"), assign the track to the primary artist (Artist A). Do not create joint artist folders.
3. **Primary Artist**: If multiple artists are listed, select only the first one for the "artist" and "albumartist" fields.
4. **Clean Tags**: Strip out junk text like "(Official Audio)", "(Lyric Video)", "[HD]", or random numbers from titles.
5. **New Path**: The new_path format MUST be: /Primary Artist/Album Name/Track Name.extension. Retain the original file extension.

Output Format:
Return ONLY a strict JSON array — no markdown code fences, no commentary, no Sources list. Write pure JSON directly to stdout.

Schema:
[
  {
    "path": "original_absolute_path",
    "title": "Correct Track Title",
    "album": "Original Studio Album",
    "artist": "Primary Artist",
    "albumartist": "Primary Artist",
    "new_path": "/Artist/Album/Track.extension"
  }
]
